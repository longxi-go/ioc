// Copyright © 2026-present the longxi-go/ioc authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compiler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWithColdVersionedReplacement(t *testing.T) {
	for _, test := range []struct {
		name, replacementPath string
	}{
		{
			name:            "same module path",
			replacementPath: "example.org/replaced",
		},
		{
			name:            "different module path",
			replacementPath: "example.org/fork",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GOWORK", "off")
			ctx := context.Background()
			cache, err := goOutput(ctx, ".", "env", "GOMODCACHE")
			if err != nil {
				t.Fatal(err)
			}
			library := t.TempDir()
			write(t, filepath.Join(library, "go.mod"), "module "+test.replacementPath+"\n\ngo 1.27.0\n")
			write(t, filepath.Join(library, "source.go"), "package dependency\nconst Source = \"replacement\"\n")
			proxy := moduleProxy(t, library, test.replacementPath, "v1.2.3")
			cachedProxy := (&url.URL{
				Scheme: "file",
				Path:   filepath.Join(strings.TrimSpace(string(cache)), "cache", "download"),
			}).String()
			t.Setenv("GOPROXY", proxy+","+cachedProxy)
			t.Setenv("GOSUMDB", "off")
			t.Setenv("GOMODCACHE", t.TempDir())
			t.Setenv("GOFLAGS", "-modcacherw")
			app := fixture(t, fixtureSource(t, "versioned-replacement/main.go.tmpl"))
			modPath := filepath.Join(app, "go.mod")
			write(t, modPath, string(mustRead(t, modPath))+fmt.Sprintf(`
require example.org/replaced v1.0.0

replace example.org/replaced v1.0.0 => %s v1.2.3
`, test.replacementPath))
			// Normal Go compilation needs the selected module's metadata but no
			// source import. Composition must also discover these cold modules.
			if output, err := goOutput(ctx, app, "test", "-mod=mod", "."); err != nil {
				t.Fatalf("valid replacement failed Go compilation: %v\n%s", err, output)
			}
			data, err := goOutput(ctx, app, "list", "-m", "-json", "example.org/replaced")
			if err != nil {
				t.Fatal(err)
			}
			var before module
			if err := json.Unmarshal(data, &before); err != nil {
				t.Fatal(err)
			}
			if before.Replace == nil || before.Replace.Dir != "" {
				t.Fatalf("replacement source must not be expanded before generation: %#v", before)
			}
			report, err := Run(ctx, Options{
				Dir: app,
			})
			if err != nil {
				t.Fatalf("Go accepts the versioned replacement but generation failed: %v", err)
			}
			if len(report.Definitions) != 1 {
				t.Fatalf("application definitions: %#v", report.Definitions)
			}
			modules, err := selectedModules(ctx, app)
			if err != nil {
				t.Fatal(err)
			}
			var selected module
			for _, dependency := range modules {
				if dependency.Path == "example.org/replaced" {
					selected = dependency
				}
			}
			if selected.Path != "example.org/replaced" || selected.Version != "v1.0.0" {
				t.Fatalf("replacement changed the logical module identity: %#v", selected)
			}
			if got := string(mustRead(t, filepath.Join(selected.Dir, "source.go"))); got != "package dependency\nconst Source = \"replacement\"\n" {
				t.Fatalf("selected module did not use replacement source: %s", got)
			}
			if output, err := goOutput(ctx, app, "test", "./..."); err != nil {
				t.Fatalf("generated application compilation: %v\n%s", err, output)
			}
		})
	}
}

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
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	http "github.com/longxi-go/ioc/pkg/codegen/http"
)

type sourceHTTPBackend struct {
	source string
}

func (b sourceHTTPBackend) Generate(http.Package) ([]byte, error) {
	return []byte(b.source), nil
}

func TestHTTPFragmentImportComposition(t *testing.T) {
	for _, tt := range []struct {
		name, source, backend, want string
	}{
		{
			"default",
			"package main\n",
			fixtureSource(t, "http-imports/default.go.tmpl"),
			"ok",
		},
		{
			"shared path",
			"package main\nimport \"time\"\n// +ioc:bean\nfunc Clock() time.Time {return time.Time{}}\n",
			fixtureSource(t, "http-imports/shared.go.tmpl"),
			"1970",
		},
		{
			"cross-path and local shadow",
			"package main\nimport \"time\"\n// +ioc:bean\nfunc Clock() time.Time {return time.Time{}}\n",
			fixtureSource(t, "http-imports/collision.go.tmpl"),
			"okshadow",
		},
		{
			"predeclared alias",
			"package main\n",
			fixtureSource(t, "http-imports/predeclared.go.tmpl"),
			"ok",
		},
		{
			"default declared package name",
			"package main\n",
			fixtureSource(t, "http-imports/declared-name.go.tmpl"),
			"target",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := fixture(t, tt.source+"// +ioc:http:get=/x\nfunc route() {}\nfunc main() {}\n")
			write(t, filepath.Join(dir, "version/v3/value.go"), "package declared\nconst Value=\"target\"\n")
			options := Options{
				Dir: dir,
				HTTP: sourceHTTPBackend{
					source: tt.backend,
				},
				Inspect: true,
			}
			if _, err := Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			options.Inspect = false
			if _, err := Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			options.Check = true
			if _, err := Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(dir, "import_test.go"), "package main\nimport \"testing\"\nfunc TestImportBehavior(t *testing.T) {if got:=HTTPGenerated();got!="+strconv.Quote(tt.want)+" {t.Fatalf(\"got %q\",got)}}\n")
			cmd := exec.Command("go", "test", ".")
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("composed imports: %v\n%s", err, out)
			}
		})
	}
}

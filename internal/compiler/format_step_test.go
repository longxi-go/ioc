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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatGenerationExpandsLayout(t *testing.T) {
	g := generation{
		module: module{
			GoVersion: "1.13.0",
		},
		outputs: map[string][]byte{
			"zz_ioc.gen.go": []byte("package fixture\ntype Config struct{ Port int }\nfunc Read() int {return Config{Port:1}.Port}\n"),
		},
	}
	if err := formatGeneration(context.Background(), &g); err != nil {
		t.Fatal(err)
	}
	source := string(g.outputs["zz_ioc.gen.go"])
	for _, fragment := range []string{"struct {\n\tPort int\n}", "func Read() int {\n", "Config{\n\t\tPort: 1,\n\t}"} {
		if !strings.Contains(source, fragment) {
			t.Errorf("missing structural layout %q:\n%s", fragment, source)
		}
	}
}

func TestFormatGenerationUsesTargetModuleVersion(t *testing.T) {
	for _, test := range []struct {
		version string
		want    string
	}{
		{
			version: "1.12",
			want:    "const Mode = 0755",
		},
		{
			version: "1.13.0",
			want:    "const Mode = 0o755",
		},
	} {
		t.Run(test.version, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.org/format\ngo "+test.version+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			selected, err := loadModule(context.Background(), dir)
			if err != nil {
				t.Fatal(err)
			}
			g := generation{
				module: selected,
				outputs: map[string][]byte{
					"zz_ioc.gen.go": []byte("package fixture\nconst Mode = 0755\n"),
				},
			}
			if err := formatGeneration(context.Background(), &g); err != nil {
				t.Fatal(err)
			}
			if got := string(g.outputs["zz_ioc.gen.go"]); !strings.Contains(got, test.want) {
				t.Fatalf("want %q, got %s", test.want, got)
			}
		})
	}
}

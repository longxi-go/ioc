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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	http "github.com/longxi-go/ioc/pkg/codegen/http"
)

type bootstrapHTTPBackend struct {
	output string
}

func (b bootstrapHTTPBackend) Bootstrap(p http.Package) ([]byte, error) {
	return []byte("package " + p.Name + "\nfunc HTTPGenerated() int { return 0 }\n"), nil
}

func (b bootstrapHTTPBackend) Generate(p http.Package) ([]byte, error) {
	return []byte("package " + p.Name + "\n" + b.output), nil
}

func TestHTTPBootstrapDeclarationsAreTemporary(t *testing.T) {
	for _, test := range []struct {
		name, output, want string
	}{
		{
			"typed application call",
			"func HTTPGenerated() int { return 42 }\n",
			"",
		},
		{
			"missing final declaration",
			"",
			"undefined: HTTPGenerated",
		},
		{
			"wrong final declaration",
			"func HTTPGenerated() string { return \"wrong\" }\n",
			"cannot use",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := fixture(t, fixtureSource(t, "http-bootstrap/main.go.tmpl"))
			options := Options{
				Dir: dir,
				HTTP: bootstrapHTTPBackend{
					output: test.output,
				},
				Inspect: true,
			}
			_, err := Run(context.Background(), options)
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("final validation: %v", err)
				}
				if _, stat := os.Stat(filepath.Join(dir, generatedFileName)); !os.IsNotExist(stat) {
					t.Fatalf("failed generation wrote output: %v", stat)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, stat := os.Stat(filepath.Join(dir, generatedFileName)); !os.IsNotExist(stat) {
				t.Fatalf("inspect wrote output: %v", stat)
			}
			options.Inspect = false
			if _, err := Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			options.Check = true
			if _, err := Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".")
			command.Dir = dir
			if data, err := command.CombinedOutput(); err != nil {
				t.Fatalf("generated execution: %v\n%s", err, data)
			}
		})
	}
}

func TestHTTPBootstrapPreservesUnownedDestination(t *testing.T) {
	for _, mode := range []string{"Inspect", "Check", "Run"} {
		t.Run(mode, func(t *testing.T) {
			dir := fixture(t, fixtureSource(t, "http-bootstrap/main.go.tmpl"))
			path := filepath.Join(dir, generatedFileName)
			original := "package main\nvar broken int = \"wrong\"\n"
			write(t, path, original)
			options := Options{
				Dir:     dir,
				Inspect: mode == "Inspect",
				Check:   mode == "Check",
				HTTP: bootstrapHTTPBackend{
					output: "func HTTPGenerated() int { return 42 }\n",
				},
			}
			_, err := Run(context.Background(), options)
			if err == nil || !strings.Contains(err.Error(), "unowned") {
				t.Errorf("unowned source must fail before bootstrap masks it: %v", err)
			}
			if got := string(mustRead(t, path)); got != original {
				t.Fatalf("unowned source changed: %q", got)
			}
		})
	}
}

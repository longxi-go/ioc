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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	http "github.com/longxi-go/ioc/pkg/codegen/http"
)

type httpControllerFixture struct {
	Route string
}

type httpTestBackend struct {
	packages []http.Package
	failure  bool
}

func (b *httpTestBackend) Generate(p http.Package) ([]byte, error) {
	b.packages = append(b.packages, p)
	if b.failure {
		return nil, fmt.Errorf("backend rejected handler")
	}
	return []byte("package " + p.Name + "\nfunc HTTPGenerated() {}\n"), nil
}

func TestHTTPBackendModelAndGeneration(t *testing.T) {
	for _, route := range []string{"+ioc:http:get=/healthz/", "+ioc:http:route:method=GET,path=/healthz/"} {
		t.Run(route, func(t *testing.T) {
			dir := fixture(t, fixtureTemplate(t, "http-controller/main.go.tmpl", httpControllerFixture{
				Route: route,
			}))
			backend := &httpTestBackend{}
			options := Options{
				Dir:  dir,
				HTTP: backend,
			}
			if _, err := Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			if len(backend.packages) != 1 {
				t.Fatalf("packages: %+v", backend.packages)
			}
			p := backend.packages[0]
			if !p.Entry || len(p.Packages) != 0 || len(p.Controllers) != 1 || len(p.Handlers) != 1 {
				t.Fatalf("model: %+v", p)
			}
			c := p.Controllers[0]
			if c.Type != "*HealthController" || c.Definition.Name != "health" || strings.Join(c.Definition.Aliases, ",") != "probe" || len(c.Routes) != 1 {
				t.Fatalf("controller: %+v", c)
			}
			r := c.Routes[0]
			if r.Path != "/api/v1/healthz/" || r.Method != "GET" || r.Handler != "health" || r.Signature == nil {
				t.Fatalf("route: %+v", r)
			}
			before := mustRead(t, filepath.Join(dir, generatedFileName))
			options.Check = true
			if _, err := Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			if string(before) != string(mustRead(t, filepath.Join(dir, generatedFileName))) {
				t.Fatal("check changed output")
			}
			options.Check = false
			options.Inspect = true
			if _, err := Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHTTPInvalidDeclarations(t *testing.T) {
	cases := []struct {
		name, source, want string
	}{
		{
			"scope",
			"// +ioc:controller\n// +ioc:scope=prototype\ntype C struct{}\n// +ioc:http:get=/x\nfunc (*C) x() {}",
			"scope",
		},
		{
			"exclusive",
			"// +ioc:controller\n// +ioc:component\ntype C struct{}",
			"exclusive",
		},
		{
			"base method",
			"// +ioc:controller\n// +ioc:http:route:method=GET,path=/x\ntype C struct{}",
			"base",
		},
		{
			"missing method",
			"// +ioc:http:route:path=/x\nfunc x() {}",
			"method",
		},
		{
			"unmanaged receiver",
			"type C struct{}\n// +ioc:http:get=/x\nfunc (*C) x() {}",
			"Controller",
		},
		{
			"unknown option",
			"// +ioc:http:route:method=GET,path=/x,runtime=true\nfunc x() {}",
			"unknown",
		},
		{
			"duplicate",
			"// +ioc:http:get=/x\n// +ioc:http:post=/x\nfunc x() {}",
			"one HTTP",
		},
		{
			"ambiguous path",
			"// +ioc:http:get=/a//b\nfunc x() {}",
			"path",
		},
		{
			"misplaced",
			"var X=1 // +ioc:http:get=/x",
			"misplaced",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dir := fixture(t, "package main\n"+tt.source+"\nfunc main(){}\n")
			_, err := Run(context.Background(), Options{
				Dir:  dir,
				HTTP: &httpTestBackend{},
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want %q: %v", tt.want, err)
			}
			if _, err := os.Stat(filepath.Join(dir, generatedFileName)); !os.IsNotExist(err) {
				t.Fatalf("unexpected output: %v", err)
			}
		})
	}
}

func TestHTTPBackendFailureAndLibrary(t *testing.T) {
	for _, library := range []bool{false, true} {
		t.Run(fmt.Sprint(library), func(t *testing.T) {
			dir := fixture(t, "package main\n// +ioc:http:get=/x\nfunc x() {}\nfunc main() {}\n")
			_, err := Run(context.Background(), Options{
				Dir:     dir,
				Library: library,
				HTTP: &httpTestBackend{
					failure: true,
				},
			})
			want := "backend rejected"
			if library {
				want = "library"
			}
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want %s: %v", want, err)
			}
			if _, err := os.Stat(filepath.Join(dir, generatedFileName)); !os.IsNotExist(err) {
				t.Fatalf("unexpected output: %v", err)
			}
		})
	}
}

func TestHTTPRouteOnlyPackagesAndFreshInspect(t *testing.T) {
	dir := fixture(t, "//go:build httpfixture\n\npackage main\nfunc main() {}\n")
	write(t, filepath.Join(dir, "handlers/handlers.go"), "package handlers\n// +ioc:http:get=/x\nfunc x() {}\n")
	backend := &httpTestBackend{}
	options := Options{
		Dir:     dir,
		Tags:    []string{"httpfixture"},
		HTTP:    backend,
		Inspect: true,
	}
	if _, err := Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, generatedFileName)); !os.IsNotExist(err) {
		t.Fatalf("inspect wrote output: %v", err)
	}
	if len(backend.packages) != 2 {
		t.Fatalf("packages: %+v", backend.packages)
	}
	for _, p := range backend.packages {
		if p.Entry && strings.Join(p.Packages, ",") != "example.org/app/handlers" {
			t.Fatalf("aggregation: %+v", p)
		}
		if !p.Entry && (len(p.Handlers) != 1 || len(p.Packages) != 0) {
			t.Fatalf("local routes: %+v", p)
		}
	}
	options.Inspect = false
	if _, err := Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	local := mustRead(t, filepath.Join(dir, "handlers", generatedFileName))
	if !strings.HasPrefix(string(local), generatedHeader) {
		t.Fatal("route-only output lacks compiler-owned generated notice")
	}
	options.Check = true
	if _, err := Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mustRead(t, filepath.Join(dir, generatedFileName))), "//go:build httpfixture") {
		t.Fatal("HTTP source combination lost build constraints")
	}
}

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
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestLicenseHeadersUseTargetModuleAndKeepGeneratedOwnership(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "codequality/HEADER"), "Copyright © 2026 the application authors.\n")
	path := filepath.Join(dir, "zz_ioc.gen.go")
	metadata := filepath.Join(dir, "ioc/module.json")
	g := generation{
		module: module{
			Dir: dir,
		},
		outputs: map[string][]byte{
			path:     []byte(generatedHeader + "\npackage application\n"),
			metadata: []byte("{}"),
		},
	}
	if err := licenseGeneration(context.Background(), &g); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(g.outputs[path], []byte(generatedHeader)) || !bytes.Contains(g.outputs[path], []byte("the application authors")) {
		t.Fatalf("missing target notice or ownership marker:\n%s", g.outputs[path])
	}
	if string(g.outputs[metadata]) != "{}" {
		t.Fatal("metadata was modified")
	}
	before := bytes.Clone(g.outputs[path])
	if err := licenseGeneration(context.Background(), &g); err != nil || !bytes.Equal(before, g.outputs[path]) {
		t.Fatal("not idempotent", err)
	}
}

func TestLicenseHeadersAreOptionalButInvalidTemplatesFail(t *testing.T) {
	dir := t.TempDir()
	g := generation{
		module: module{
			Dir: dir,
		},
		outputs: map[string][]byte{
			"zz_ioc.gen.go": []byte(generatedHeader + "\npackage application\n"),
		},
	}
	if err := licenseGeneration(context.Background(), &g); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "codequality/HEADER"), "invalid template")
	if err := licenseGeneration(context.Background(), &g); err == nil {
		t.Fatal("invalid target template accepted")
	}
}

func TestLibraryPublicationWithSourceHeaders(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "codequality/HEADER"), "Copyright © 2026 the library authors.\n")
	options := Options{
		Dir:     library,
		Library: true,
	}
	if _, err := Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	options.Check = true
	if _, err := Run(context.Background(), options); err != nil {
		t.Fatal("library header regeneration is stale", err)
	}
	for _, name := range []string{generatedFileName, indexSourcePath} {
		data := mustRead(t, filepath.Join(library, name))
		if !bytes.HasPrefix(data, []byte(generatedHeader)) || !bytes.Contains(data, []byte("the library authors.")) {
			t.Fatalf("missing library notice in %s:\n%s", name, data)
		}
	}
	app := fixture(t, "package main\nfunc main() {}\n")
	write(t, filepath.Join(app, "codequality/HEADER"), "Copyright © 2026 the application authors.\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal("published hashes do not match licensed artifacts", err)
	}
	data := string(mustRead(t, filepath.Join(app, generatedFileName)))
	if !strings.Contains(data, "the application authors.") || strings.Contains(data, "the library authors.") {
		t.Fatalf("library notice leaked into consumer:\n%s", data)
	}
}

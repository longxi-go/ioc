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
	"archive/zip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// coldZIPFixture supplies the unique module identity used by the local proxy.
type coldZIPFixture struct {
	ModulePath string
}

func moduleProxy(t *testing.T, library, path, version string) string {
	t.Helper()
	proxy := t.TempDir()
	cache := filepath.Join(proxy, filepath.FromSlash(path), "@v")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(cache, "list"), version+"\n")
	write(t, filepath.Join(cache, version+".info"), fmt.Sprintf(`{"Version":%q,"Time":"2026-11-18T00:00:00Z"}`, version))
	write(t, filepath.Join(cache, version+".mod"), string(mustRead(t, filepath.Join(library, "go.mod"))))
	f, err := os.Create(filepath.Join(cache, version+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(f)
	err = filepath.WalkDir(library, func(name string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(library, name)
		if err != nil {
			return err
		}
		entry, err := writer.Create(path + "@" + version + "/" + filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		_, err = entry.Write(mustRead(t, name))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return "file://" + proxy
}

func TestColdZIPDiscoveryAndVendor(t *testing.T) {
	library, _ := libraryFixture(t)
	path := fmt.Sprintf("example.org/coldioc%d", time.Now().UnixNano())
	write(t, filepath.Join(library, "go.mod"), strings.ReplaceAll(string(mustRead(t, filepath.Join(library, "go.mod"))), "example.org/library", path))
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "cold-zip/library.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPROXY", moduleProxy(t, library, path, "v1.2.3"))
	t.Setenv("GOSUMDB", "off")
	app := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\n")
	if report, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil || len(report.Definitions) != 3 {
		t.Fatalf("cold selected module without source import: %#v %v", report, err)
	}
	write(t, filepath.Join(app, "events_test.go"), fixtureTemplate(t, "cold-zip/runtime_test.go.tmpl", coldZIPFixture{
		ModulePath: path,
	}))
	verifyEvents := func() {
		t.Helper()
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = app
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("downloaded/vendor event execution: %v %s", err, out)
		}
	}
	verifyEvents()
	// Dependency source and adapters remain unchanged after consumer generation.
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
		Check:   true,
	}); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = app
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("vendor: %v %s", err, out)
	}
	// Go vendor retains Go-visible Metadata but drops ancillary JSON. Consumers
	// must recover the exact same manifest from that public index.
	if _, err := os.Stat(filepath.Join(app, "vendor", path, "ioc/module.json")); !os.IsNotExist(err) {
		t.Fatal("fixture did not reproduce dropped JSON")
	}
	if report, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil || len(report.Definitions) != 3 {
		t.Fatalf("retained vendor metadata: %#v %v", report, err)
	}
	verifyEvents()
	if err := os.Remove(filepath.Join(app, "vendor", path, "iocindex/index.gen.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err == nil {
		t.Fatal("accepted incomplete vendor metadata")
	}
}

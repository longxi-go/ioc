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

package codegen

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/longxi-go/ioc/pkg/cli"

	core "github.com/longxi-go/ioc/pkg/codegen"
	"golang.org/x/mod/modfile"
)

func consumer(t *testing.T, scenario string) string {
	t.Helper()
	t.Setenv("GOWORK", "off")
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.AddModuleStmt("example.org/consumer"); err != nil {
		t.Fatal(err)
	}
	if err = m.AddRequire("github.com/longxi-go/ioc/adaptors/gin", "v0.2.0"); err != nil {
		t.Fatal(err)
	}
	if err = m.AddReplace("github.com/longxi-go/ioc/adaptors/gin", "", root, ""); err != nil {
		t.Fatal(err)
	}
	if err = m.AddReplace("github.com/longxi-go/ioc", "", filepath.Clean(filepath.Join(root, "../..")), ""); err != nil {
		t.Fatal(err)
	}
	if work := os.Getenv("IOC_TEST_WORKSPACE"); work != "" {
		data, err = os.ReadFile(work)
		if err != nil {
			t.Fatal(err)
		}
		w, err := modfile.ParseWork(work, data, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, u := range w.Use {
			p := u.Path
			if !filepath.IsAbs(p) {
				p = filepath.Join(filepath.Dir(work), p)
			}
			b, err := os.ReadFile(filepath.Join(p, "go.mod"))
			if err != nil {
				t.Fatal(err)
			}
			dep, err := modfile.Parse("go.mod", b, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err = m.AddReplace(dep.Module.Mod.Path, "", p, ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	data, err = m.Format()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(filepath.Join("testdata", scenario), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(filepath.Join("testdata", scenario), path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, strings.TrimSuffix(relative, ".tmpl"))
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tidy: %v\n%s", err, out)
	}
	return dir
}

func TestCompiledConsumer(t *testing.T) {
	dir := consumer(t, "application")
	options := core.Options{
		Dir:  dir,
		HTTP: Adapter{},
	}
	if _, err := core.Inspect(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "zz_ioc.gen.go")); !os.IsNotExist(err) {
		t.Fatal("inspect wrote output")
	}
	if _, err := core.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	options.Check = true
	if _, err := core.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	for _, verb := range []string{"check", "explain", "graph"} {
		var stdout, stderr bytes.Buffer
		if code := cli.Run(context.Background(), []string{verb, "--dir", dir}, &stdout, &stderr, cli.Options{
			Name: "ginctl",
			HTTP: Adapter{},
		}); code != 0 {
			t.Fatalf("%s: %d %s", verb, code, stderr.String())
		}
		if verb != "check" && !json.Valid(stdout.Bytes()) {
			t.Fatalf("%s emitted non-JSON output: %s", verb, stdout.String())
		}
	}

	cmd := exec.Command("go", "test", "-race", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("consumer: %v\n%s", err, out)
	}
}

func TestInvalidSignatures(t *testing.T) {
	for _, signature := range []string{"func endpoint() {}", "func endpoint(c *gin.Context) error {return nil}", "func endpoint(c gin.Context) {}", "func endpoint(c ...*gin.Context) {}"} {
		dir := consumer(t, "invalid")
		data := []byte("package main\nimport \"github.com/gin-gonic/gin\"\nvar _ *gin.Context\n// +ioc:http:get=/\n" + signature + "\nfunc main(){}\n")
		if err := os.WriteFile(filepath.Join(dir, "main.go"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := core.Run(context.Background(), core.Options{
			Dir:  dir,
			HTTP: Adapter{},
		}); err == nil || !strings.Contains(err.Error(), "main.go:") {
			t.Fatalf("signature %s: %v", signature, err)
		}
	}
}

func TestInvalidWildcards(t *testing.T) {
	for _, path := range []string{"/:id:name", "/files/*tail/more", "/files*tail", "/:"} {
		dir := consumer(t, "invalid")
		source := "package main\nimport \"github.com/gin-gonic/gin\"\n// +ioc:http:get=" + strconv.Quote(path) + "\nfunc endpoint(*gin.Context){}\nfunc main(){}\n"
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := core.Run(context.Background(), core.Options{
			Dir:  dir,
			HTTP: Adapter{},
		}); err == nil || !strings.Contains(err.Error(), "main.go:") || !strings.Contains(err.Error(), "Gin route") {
			t.Fatalf("path %s: %v", path, err)
		}
	}
}

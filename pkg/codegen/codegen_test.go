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
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/longxi-go/ioc/internal/testmodule"
)

func TestRunRejectsMissingModule(t *testing.T) {
	if _, err := Run(context.Background(), Options{
		Dir: t.TempDir(),
	}); err == nil {
		t.Fatal("accepted directory without module")
	}
}

func TestRunAndInspectPublicFacade(t *testing.T) {
	dir := publicFixture(t)
	report, err := Inspect(context.Background(), Options{
		Dir: dir,
	})
	if err != nil || len(report.Definitions) != 1 || len(report.Files) != 1 {
		t.Fatalf("inspect: %#v %v", report, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "zz_ioc.gen.go")); !os.IsNotExist(err) {
		t.Fatal("Inspect wrote source")
	}
	report.Definitions[0].Name = "mutated detached report"
	report, err = Run(context.Background(), Options{
		Dir: dir,
	})
	if err != nil || report.Definitions[0].Name != "Service" {
		t.Fatal(report, err)
	}
	if _, err := Run(context.Background(), Options{
		Dir:   dir,
		Check: true,
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, Options{
		Dir: dir,
	}); err == nil {
		t.Fatal("ignored cancellation")
	}
}

func TestFixturesFromStandaloneSourceCopy(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "ioc source")
	copyStandaloneSource(t, root, dir)
	if err := testmodule.UseWorkspace(dir); err != nil {
		t.Fatal(err)
	}

	// Run each fixture builder from a checkout with no sibling modules. The
	// space in its path also exercises Go module replacement-path quoting.
	// An explicit test workspace selects local contracts in this copy only.
	tests := []string{
		"TestGenerateAndCheckApplication",
		"TestGenerateCheckExplainAndGraph",
		"TestRunAndInspectPublicFacade",
		"TestGeneratedConditionsComposeWithoutExecutingFactories",
	}
	cmd := exec.Command("go", "test", "-count=1",
		"./internal/compiler", "./internal/command", "./pkg/codegen", "./integration",
		"-run", "^("+strings.Join(tests, "|")+")$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standalone source fixtures: %v\n%s", err, output)
	}
}

func copyStandaloneSource(t *testing.T, root, dir string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if relative == "." {
				return nil
			}
			if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "docs" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		source := strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".tmpl")
		metadata := relative == "go.mod" || relative == "go.sum" || relative == filepath.Join("codequality", "HEADER")
		if !source && !metadata {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		destination := filepath.Join(dir, relative)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func publicFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := testmodule.Write(dir, "example.org/facade", root); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join("testdata", "public", "main.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	writePublicFixture(t, dir, "main.go", source)
	return dir
}

func writePublicFixture(t *testing.T, dir, name string, source []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), source, 0o644); err != nil {
		t.Fatal(err)
	}
}

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
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/longxi-go/ioc/internal/testmodule"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/imports"
)

func fixture(t *testing.T, source string) string {
	t.Helper()
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := testmodule.Write(dir, "example.org/app", root); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "main.go"), source)
	return dir
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateAndCheckApplication(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "application/main.go.tmpl"))
	report, err := Run(context.Background(), Options{
		Dir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Definitions) != 2 {
		t.Fatalf("definitions: %#v", report)
	}
	data, err := os.ReadFile(filepath.Join(dir, "zz_ioc.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"catalog.Default().Contribute", "catalog.Default().Install", "bean.Assign", ".port =", ".Init(ctx)", "case *Port:"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s:\n%s", want, data)
		}
	}
	before := string(data)
	// Saving through an import-aware formatter must not make generated output stale.
	saved, err := imports.Process(filepath.Join(dir, "zz_ioc.gen.go"), data, &imports.Options{
		FormatOnly: true,
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
	})
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "zz_ioc.gen.go"), string(saved))
	if _, err := Run(context.Background(), Options{
		Dir:   dir,
		Check: true,
	}); err != nil {
		t.Fatalf("save-time import formatting made generated output stale: %v", err)
	}
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(dir, "zz_ioc.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != before {
		t.Fatal("nondeterministic output")
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated compilation: %v\n%s", err, out)
	}
	write(t, filepath.Join(dir, "main.go"), strings.ReplaceAll(string(mustRead(t, filepath.Join(dir, "main.go"))), "+ioc:primary", "+ioc:lazy"))
	if _, err := Run(context.Background(), Options{
		Dir:   dir,
		Check: true,
	}); err == nil {
		t.Fatal("check accepted stale output")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestGeneratedMetadataFieldsUseSeparateLines(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "metadata/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "zz_ioc.gen.go")
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, path, mustRead(t, path), 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(file.Decls); i++ {
		previous := positions.Position(file.Decls[i-1].End()).Line
		current := positions.Position(file.Decls[i].Pos()).Line
		if current-previous < 2 {
			t.Fatalf("generated top-level declarations need a separating blank line at %s", positions.Position(file.Decls[i].Pos()))
		}
	}
	checked := 0
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok || len(literal.Elts) < 3 {
			return true
		}
		if _, keyed := literal.Elts[0].(*ast.KeyValueExpr); !keyed {
			return true
		}
		checked++
		previous := positions.Position(literal.Lbrace).Line
		for _, field := range literal.Elts {
			position := positions.Position(field.Pos())
			if position.Line <= previous {
				t.Fatalf("generated metadata fields share a line at %s", position)
			}
			previous = positions.Position(field.End()).Line
		}
		if positions.Position(literal.Rbrace).Line <= previous {
			t.Fatalf("generated metadata closing brace shares a field line at %s", positions.Position(literal.Rbrace))
		}
		return true
	})
	if checked == 0 {
		t.Fatal("generated fixture did not exercise metadata layout")
	}
}

func TestGeneratedContextParametersRemainOperationScoped(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "context-injection/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "context_test.go"), fixtureSource(t, "context-injection/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("context injection: %v\n%s", err, output)
	}
}

func TestRejectedAnnotationsAndTypes(t *testing.T) {
	for _, tc := range []struct {
		name, fixture, want string
	}{
		{
			"unknown",
			"rejected/unknown.go.tmpl",
			"unknown",
		},
		{
			"generic",
			"rejected/generic.go.tmpl",
			"generic",
		},
		{
			"method",
			"rejected/method.go.tmpl",
			"method",
		},
		{
			"signature",
			"rejected/signature.go.tmpl",
			"result",
		},
		{
			"missing",
			"rejected/missing.go.tmpl",
			"missing",
		},
		{
			"cycle",
			"rejected/cycle.go.tmpl",
			"cycle",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := fixture(t, fixtureSource(t, tc.fixture))
			_, err := Run(context.Background(), Options{
				Dir: dir,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %s, got %v", tc.want, err)
			}
			if _, e := os.Stat(filepath.Join(dir, "zz_ioc.gen.go")); !os.IsNotExist(e) {
				t.Fatal("failed analysis wrote adapter")
			}
		})
	}
}

func TestNestedProviderDiagnosesWithoutEagerConstruction(t *testing.T) {
	for _, name := range []string{"optional", "slice", "provider"} {
		dir := fixture(t, fixtureSource(t, "nested-provider/"+name+".go.tmpl"))
		if _, err := Run(context.Background(), Options{
			Dir: dir,
		}); err == nil || !strings.Contains(err.Error(), "Provider[Optional[T]]") {
			t.Fatalf("accepted %s: %v", name, err)
		}
	}
}

func TestGeneratedWiringExecutesPrivateFieldsAndNestedProvider(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "private-wiring/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "wiring_test.go"), fixtureSource(t, "private-wiring/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated runtime behavior: %v\n%s", err, out)
	}
}

func TestAnonymousContractCanonicalizationAndStrictMarkerPlacement(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "anonymous-contracts/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal("equivalent anonymous contracts", err)
	}
	for _, source := range []string{"// +ioc:unknown\npackage main\nfunc main(){}", fixtureSource(t, "anonymous-contracts/misplaced-marker.go.tmpl")} {
		dir := fixture(t, source)
		if _, err := Run(context.Background(), Options{
			Dir: dir,
		}); err == nil {
			t.Fatalf("silently ignored marker: %s", source)
		}
	}
}

func TestInspectAndCheckNeverMutateModuleMetadata(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=mod")
	dir := fixture(t, "package main\nimport _ \"golang.org/x/tools/go/packages\"\nfunc main(){}\n")
	// Leave the directly imported tooling dependency transitive so ordinary Go
	// loading would update both go.mod and the missing go.sum.
	module, err := modfile.Parse("go.mod", mustRead(t, filepath.Join(dir, "go.mod")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := module.DropRequire("golang.org/x/tools"); err != nil {
		t.Fatal(err)
	}
	data, err := module.Format()
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "go.mod"), string(data))
	if err := os.Remove(filepath.Join(dir, "go.sum")); err != nil {
		t.Fatal(err)
	}
	original := string(mustRead(t, filepath.Join(dir, "go.mod")))
	_, err = Run(context.Background(), Options{
		Dir:     dir,
		Inspect: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, filepath.Join(dir, "go.mod"))) != original {
		t.Fatal("Inspect modified go.mod")
	}
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); !os.IsNotExist(err) {
		t.Fatal("Inspect wrote go.sum")
	}
	_, err = Run(context.Background(), Options{
		Dir:   dir,
		Check: true,
	})
	if err == nil {
		t.Fatal("Check accepted missing generated output")
	}
	if string(mustRead(t, filepath.Join(dir, "go.mod"))) != original {
		t.Fatal("Check modified go.mod")
	}
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); !os.IsNotExist(err) {
		t.Fatal("Check wrote go.sum")
	}
}

func TestConstructorParameterSelectors(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "parameter-selectors/main.go.tmpl"))
	report, err := Run(context.Background(), Options{
		Dir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, d := range report.Definitions {
		if strings.HasSuffix(d.ID, ".Pair") {
			found = true
			if d.Dependencies[0].Selector.Name != "readDB" || d.Dependencies[1].Selector.Name != "writeDB" {
				t.Fatal(d)
			}
		}
	}
	if !found {
		t.Fatal("missing pair")
	}
}

func TestReadOnlyMirrorsPreserveEmbeddedAssets(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "embedded-assets/main.go.tmpl"))
	write(t, filepath.Join(dir, "asset.txt"), "embedded asset")
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	for _, o := range []Options{
		{
			Dir:   dir,
			Check: true,
		},
		{
			Dir:     dir,
			Inspect: true,
		},
	} {
		if _, err := Run(context.Background(), o); err != nil {
			t.Fatal("embed source mirror", err)
		}
	}
}

func TestEquivalentGoTypeIdentities(t *testing.T) {
	for _, tc := range []struct {
		name, fixture string
	}{
		{
			"builtin-byte",
			"type-identities/builtin-byte.go.tmpl",
		},
		{
			"builtin-rune",
			"type-identities/builtin-rune.go.tmpl",
		},
		{
			"interface-embedding",
			"type-identities/interface-embedding.go.tmpl",
		},
		{
			"recursive-alias",
			"type-identities/recursive-alias.go.tmpl",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := fixture(t, fixtureSource(t, tc.fixture))
			if _, err := Run(context.Background(), Options{
				Dir: dir,
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestParameterSelectorDiagnostics(t *testing.T) {
	for _, name := range []string{"missing-parameter", "missing-selector", "conflicting-selectors", "duplicate-qualifier"} {
		dir := fixture(t, fixtureSource(t, "parameter-diagnostics/"+name+".go.tmpl"))
		if _, err := Run(context.Background(), Options{
			Dir: dir,
		}); err == nil {
			t.Fatal("accepted invalid selector", name)
		}
	}
}

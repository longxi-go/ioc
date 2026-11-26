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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func libraryFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := fixture(t, `package library`)
	mod := string(mustRead(t, filepath.Join(dir, "go.mod")))
	write(t, filepath.Join(dir, "go.mod"), strings.ReplaceAll(mod, "example.org/app", "example.org/library"))
	write(t, filepath.Join(dir, "main.go"), `package library
// +ioc:component
type Public struct{}
`)
	write(t, filepath.Join(dir, "internal/hidden/hidden.go"), `package hidden
// +ioc:component
type Secret struct{}
`)
	write(t, filepath.Join(dir, "private/private.go"), `package private
// +ioc:bean
func New() string {return "private"}
`)
	return dir, "example.org/library"
}

func TestLibraryPublicationConsumerAndFreshness(t *testing.T) {
	library, path := libraryFixture(t)
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal(err)
	}
	if data := string(mustRead(t, filepath.Join(app, "zz_ioc.gen.go"))); !strings.Contains(data, path+"/iocindex") {
		t.Fatalf("no public index import: %s", data)
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = app
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("consumer: %v %s", err, out)
	}
	manifest := mustRead(t, filepath.Join(library, "ioc/module.json"))
	var pub publication
	if err := json.Unmarshal(manifest, &pub); err != nil {
		t.Fatal(err)
	}
	if len(pub.Recipes) != 3 {
		t.Fatalf("missing recipes: %#v", pub)
	}
	write(t, filepath.Join(library, "private/private.go"), string(mustRead(t, filepath.Join(library, "private/private.go")))+"\n// fresh edit\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err == nil || !strings.Contains(err.Error(), "freshness") {
		t.Fatalf("stale publication accepted: %v", err)
	}
}

func TestBuildVariantsAndStaleOwnedImports(t *testing.T) {
	dir := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(dir, "service.go"), fixtureSource(t, "build-variants/normal.go.tmpl"))
	write(t, filepath.Join(dir, "service_special.go"), fixtureSource(t, "build-variants/special.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	if data := string(mustRead(t, filepath.Join(dir, "zz_ioc.gen.go"))); !strings.Contains(data, "//go:build") || !strings.Contains(data, "!special") {
		t.Fatalf("missing negative constraint: %s", data)
	}
	// A stale import must not prevent regeneration after a package disappears.
	write(t, filepath.Join(dir, "zz_ioc.gen.go"), generatedHeader+"package main\nimport _ \"example.invalid/deleted\"\n")
	if _, err := Run(context.Background(), Options{
		Dir:  dir,
		Tags: []string{"special"},
	}); err != nil {
		t.Fatal(err)
	}
	if data := string(mustRead(t, filepath.Join(dir, "zz_ioc.gen.go"))); strings.Contains(data, "Normal") || !strings.Contains(data, "Special") {
		t.Fatal("mixed incompatible symbols")
	}
}

func TestWorkspaceAndMultipleEntries(t *testing.T) {
	library, path := libraryFixture(t)
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main(){}\n")
	work := filepath.Join(t.TempDir(), "go.work")
	write(t, work, "go 1.27.0\nuse (\n"+app+"\n"+library+"\n)\n")
	t.Setenv("GOWORK", work)
	if report, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil || len(report.Definitions) != 3 {
		t.Fatalf("workspace foreign module %s: %#v %v", path, report, err)
	}
	t.Setenv("GOWORK", "off")
	write(t, filepath.Join(app, "cmd/other/main.go"), "package main\n// +ioc:component\ntype Other struct{}\nfunc main(){}\n")
	if report, err := Run(context.Background(), Options{
		Dir:   app,
		Entry: ".",
		Roots: []string{"."},
	}); err != nil || len(report.Definitions) != 0 {
		t.Fatalf("explicit entry included other main: %#v %v", report, err)
	}
	if _, err := os.Stat(filepath.Join(app, "cmd/other/zz_ioc.gen.go")); !os.IsNotExist(err) {
		t.Fatal("other entry was modified")
	}
}

func TestTwoEntriesKeepIndependentPlans(t *testing.T) {
	dir := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(dir, "services/service.go"), "package services\n// +ioc:component\ntype Shared struct{}\n")
	write(t, filepath.Join(dir, "cmd/other/main.go"), "package main\nfunc main(){}\n")
	for _, entry := range []string{".", "./cmd/other", ".", "./cmd/other"} {
		options := Options{
			Dir:   dir,
			Entry: entry,
			Roots: []string{"./services/..."},
		}
		if _, err := Run(context.Background(), options); err != nil {
			t.Fatal(entry, err)
		}
		options.Check = true
		if _, err := Run(context.Background(), options); err != nil {
			t.Fatal(entry, err)
		}
	}
	for _, name := range []string{"zz_ioc.gen.go", "cmd/other/zz_ioc.gen.go", "services/zz_ioc.gen.go"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(name, err)
		}
	}
}

func TestPublicationRejectsNewUnindexedPackage(t *testing.T) {
	library, path := libraryFixture(t)
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(library, "newpkg/new.go"), "package newpkg\n// +ioc:component\ntype Added struct{}\n")
	app := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err == nil {
		t.Fatal("accepted unindexed annotation package")
	}
}

func TestPrivateImplementationExposesPublicContract(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "private-contracts/library.go.tmpl"))
	write(t, filepath.Join(library, "private/private.go"), "package private\n// +ioc:bean\nfunc newPrivate()string{return \"hidden constructor\"}\n")
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal(err)
	}
	data := string(mustRead(t, filepath.Join(app, "zz_ioc.gen.go")))
	if strings.Contains(data, "privateService:") {
		t.Fatal("query leaked private implementation")
	}
	if !strings.Contains(data, ".Public") {
		t.Fatal("public contract query missing")
	}
}

func TestAmbiguousMultipleEntriesRequireRoots(t *testing.T) {
	dir := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(dir, "cmd/worker/main.go"), "package main\nfunc main(){}\n")
	write(t, filepath.Join(dir, "worker/component.go"), "package worker\n// +ioc:component\ntype Worker struct{}\n")
	if _, err := Run(context.Background(), Options{
		Dir:   dir,
		Entry: ".",
	}); err == nil {
		t.Fatal("implicit roots crossed entry boundary")
	}
}

func TestOpenLibraryPortsBindInApplication(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "open-ports/library.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal("open library must publish external ports", err)
	}
	app := fixture(t, fixtureSource(t, "open-ports/main.go.tmpl"))
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(app, "main.go"), "package main\nfunc main(){}\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err == nil {
		t.Fatal("application accepted unbound library port")
	}
}

func TestCrossPackageTagPlanExcludesIncompatibleContracts(t *testing.T) {
	dir := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(dir, "services/base.go"), "package services\n")
	write(t, filepath.Join(dir, "services/special.go"), fixtureSource(t, "tag-plan/special.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:  dir,
		Tags: []string{"special"},
	}); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cross-package generated type leaked across tags: %v %s", err, out)
	}
}

func TestWorkspaceVendorUsesWorkspaceRoot(t *testing.T) {
	library, _ := libraryFixture(t)
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main(){}\n")
	workRoot := t.TempDir()
	work := filepath.Join(workRoot, "go.work")
	write(t, work, "go 1.27.0\nuse (\n"+app+"\n"+library+"\n)\n")
	t.Setenv("GOWORK", work)
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "work", "vendor")
	cmd.Dir = app
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("work vendor: %v %s", err, out)
	}
	if _, err := Run(context.Background(), Options{
		Dir:   app,
		Check: true,
	}); err != nil {
		t.Fatal("workspace vendor check", err)
	}
}

func TestPrivateAliasAndCompositeVisibility(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "composite-visibility/library.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal(err)
	}
	data := string(mustRead(t, filepath.Join(app, "zz_ioc.gen.go")))
	if !strings.Contains(data, "M()") ||
		!strings.Contains(data, ".Public") ||
		strings.Contains(data, ".hidden") ||
		strings.Contains(data, ".privateAlias") {
		t.Fatal("incorrect visible contract representation", data)
	}
}

func TestBuildEnvironmentFlagsAndSourceTarget(t *testing.T) {
	dir := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(dir, "variant.go"), fixtureSource(t, "target-flags/normal.go.tmpl"))
	write(t, filepath.Join(dir, "variant_special.go"), fixtureSource(t, "target-flags/special.go.tmpl"))
	t.Setenv("GOFLAGS", "-tags=special")
	report, err := Run(context.Background(), Options{
		Dir: dir,
	})
	if err != nil || len(report.Definitions) != 1 || report.Definitions[0].Name != "Special" {
		t.Fatalf("GOFLAGS ignored: %#v %v", report, err)
	}
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOOS", "windows")
	t.Setenv("GOARCH", "amd64")
	write(t, filepath.Join(dir, "platform_windows.go"), "package main\n// +ioc:component\ntype Windows struct{}\n")
	write(t, filepath.Join(dir, "platform_linux.go"), "package main\n// +ioc:component\ntype Linux struct{}\n")
	report, err = Run(context.Background(), Options{
		Dir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range report.Definitions {
		if d.Name == "Linux" {
			t.Fatal("host GOOS leaked into target analysis")
		}
	}
	if data := string(mustRead(t, filepath.Join(dir, "zz_ioc.gen.go"))); !strings.Contains(data, "windows") || strings.Contains(data, "**Linux") {
		t.Fatal("wrong target constraint", data)
	}
}

func TestOpenLibraryCandidateCanBeOverriddenByApplicationPrimary(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "primary-override/library.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal("open candidate set cannot prove self-cycle", err)
	}
	app := fixture(t, fixtureSource(t, "primary-override/main.go.tmpl"))
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(app, "main.go"), "package main\nfunc main(){}\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err == nil {
		t.Fatal("closed application accepted actual self-cycle")
	}
}

func TestUnmanagedDependencyStringsAndExamplesDoNotRequirePublication(t *testing.T) {
	library := fixture(t, "package library\nconst MarkerText = \"+ioc:component\"\n")
	path := "example.org/unmanaged"
	write(t, filepath.Join(library, "go.mod"), strings.ReplaceAll(string(mustRead(t, filepath.Join(library, "go.mod"))), "example.org/app", path))
	write(t, filepath.Join(library, "examples/main.go"), "package main\n// +ioc:component\ntype Example struct{}\nfunc main(){}\n")
	app := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal("unmanaged dependency must not require publication", err)
	}
}

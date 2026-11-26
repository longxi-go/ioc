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
	"strings"
	"testing"
)

func TestLegacyLiveABIBatchCannotCompileAsCurrent(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "prerequisites/legacy.go.tmpl"))
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "unknown field Auto") {
		t.Fatalf("legacy recipe with a live ABI constant must require regeneration: %v\n%s", err, output)
	}
}

func TestGeneratedPrerequisitesExecuteAndRemainFresh(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "prerequisites/main.go.tmpl"))
	report, err := Run(context.Background(), Options{
		Dir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, edge := range report.Edges {
		if edge.Kind == "depends-on" &&
			strings.HasSuffix(edge.From, ".newRepository") &&
			strings.HasSuffix(edge.To, ".newMigration") {
			found = true
		}
	}
	if !found {
		t.Fatal("missing prerequisite graph edge", report.Edges)
	}
	write(t, filepath.Join(dir, "runtime_test.go"), fixtureSource(t, "prerequisites/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated prerequisite behavior: %v\n%s", err, output)
	}
	if _, err := Run(context.Background(), Options{
		Dir:   dir,
		Check: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPublishedLibraryPrerequisitesExecuteWithoutRewritingLibrary(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "prerequisites/library.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	adapter := string(mustRead(t, filepath.Join(library, "zz_ioc.gen.go")))
	consumer := fixture(t, "package main\nfunc main() {}\n")
	mod := string(mustRead(t, filepath.Join(consumer, "go.mod")))
	write(t, filepath.Join(consumer, "go.mod"), mod+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: consumer,
	}); err != nil {
		t.Fatal(err)
	}
	if adapter != string(mustRead(t, filepath.Join(library, "zz_ioc.gen.go"))) {
		t.Fatal("application generation rewrote library adapters")
	}
	write(t, filepath.Join(consumer, "runtime_test.go"), fixtureSource(t, "prerequisites/library_runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = consumer
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("published prerequisite execution: %v\n%s", err, output)
	}
}

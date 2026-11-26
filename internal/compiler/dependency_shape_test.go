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
)

func TestUnsupportedDependencyShapesFailBeforeOutput(t *testing.T) {
	shapes := []string{
		"gs.Optional[[]int]",
		"[]gs.Optional[int]",
		"gs.Optional[gs.Optional[int]]",
		"[][]int",
		"gs.Optional[gs.Provider[int]]",
		"[]gs.Provider[int]",
		"gs.Provider[gs.Provider[int]]",
		"gs.Provider[gs.Optional[[]int]]",
		"gs.Provider[[]gs.Optional[int]]",
	}
	for _, shape := range shapes {
		for _, field := range []bool{false, true} {
			t.Run(shape+map[bool]string{false: "/parameter", true: "/field"}[field], func(t *testing.T) {
				fixturePath := "dependency-shapes/invalid.go.tmpl"
				if field {
					fixturePath = "dependency-shapes/invalid-field.go.tmpl"
				}
				source := fixtureSource(t, fixturePath)
				source = strings.ReplaceAll(source, "DEPENDENCY", shape)
				dir := fixture(t, source)
				_, err := Run(context.Background(), Options{
					Dir: dir,
				})
				if err == nil ||
					!strings.Contains(err.Error(), "unsupported dependency shape") ||
					!strings.Contains(err.Error(), "main.go:") {
					t.Fatalf("expected source-aware unsupported shape diagnostic, got %v", err)
				}
				if _, err := os.Stat(filepath.Join(dir, "zz_ioc.gen.go")); !os.IsNotExist(err) {
					t.Fatalf("invalid shape wrote generated output: %v", err)
				}
			})
		}
	}
}

func TestGeneratedSixDependencyShapesAndExactCollections(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "dependency-shapes/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "runtime_test.go"), fixtureSource(t, "dependency-shapes/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated dependency behavior: %v\n%s", err, output)
	}
}

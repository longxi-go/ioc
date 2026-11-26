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
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestSourceInventoryIgnoresGoIgnoredFiles(t *testing.T) {
	dir := t.TempDir()
	active := filepath.Join(dir, "repository.go")
	write(t, active, "package fixture\n")
	write(t, filepath.Join(dir, "_ignored.go"), "//go:build hidden\n\npackage fixture\n")
	write(t, filepath.Join(dir, ".ignored.go"), "//go:build hidden\n\npackage fixture\n")
	pkg := &packages.Package{
		GoFiles: []string{active},
	}
	t.Run("constraints", func(t *testing.T) {
		got, err := buildConstraint(pkg)
		if err != nil || got != "" {
			t.Fatalf("ignored files changed constraint: %q, %v", got, err)
		}
	})
	t.Run("hashes", func(t *testing.T) {
		got, err := sourceHashes(pkg)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got["repository.go"] == "" {
			t.Fatalf("ignored files entered source inventory: %v", got)
		}
	})
}

func TestOwnedOverlayIgnoresGoIgnoredFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"_ignored.go", ".ignored.go"} {
		write(t, filepath.Join(dir, name), generatedHeader+"\npackage fixture\n")
	}
	got, err := ownedOverlay(dir)
	if err != nil || len(got) != 0 {
		t.Fatalf("ignored files entered owned overlay: %v, %v", got, err)
	}
}

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
	"path/filepath"
	"testing"
)

func TestSelectedVendorModulesRequirePackages(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=vendor")
	dir := t.TempDir()
	write(t, filepath.Join(dir, "go.mod"), "module example.org/app\n\ngo 1.27.0\n")
	write(t, filepath.Join(dir, "vendor/modules.txt"), string(mustRead(t, "testdata/vendor/modules.txt")))
	modules, err := selectedModules(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		path, version string
	}{
		{
			path:    "example.org/versioned",
			version: "v1.0.0",
		},
		{
			path:    "example.org/local",
			version: "v1.1.0",
		},
		{
			path:    "example.org/missing",
			version: "v1.0.0",
		},
	}
	if len(modules) != len(want) {
		t.Fatalf("selected modules include headers without packages or duplicates: %#v", modules)
	}
	for i, expected := range want {
		selected := modules[i]
		if selected.Path != expected.path || selected.Version != expected.version {
			t.Fatalf("selected logical identity: %#v; want %s@%s", selected, expected.path, expected.version)
		}
		if selected.Dir != filepath.Join(dir, "vendor", filepath.FromSlash(expected.path)) {
			t.Fatalf("replacement escaped the vendored source tree: %#v", selected)
		}
	}
	// A package declaration remains selected even when its source was removed.
	// The reader must report that corruption instead of silently skipping it.
	if _, _, err := readPublication(modules[2], target{}, nil); err == nil {
		t.Fatal("missing source for a declared vendored package was accepted")
	}
}

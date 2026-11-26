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

package testmodule

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestWriteExplicitWorkspaceDependencies(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	contracts := filepath.Join(root, "contracts")
	for _, dir := range []string{source, contracts} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(source, "go.mod"):    "module example.org/source\n\ngo 1.27.0\n\nrequire example.org/contracts v1.2.3\n",
		filepath.Join(source, "go.sum"):    "",
		filepath.Join(contracts, "go.mod"): "module example.org/contracts\n\ngo 1.27.0\n",
		filepath.Join(root, "go.work"):     "go 1.27.0\n\nuse (\n ./source\n ./contracts\n)\n",
	}
	for path, text := range files {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("IOC_TEST_WORKSPACE", filepath.Join(root, "go.work"))
	dir := t.TempDir()
	if err := Write(dir, "example.org/consumer", source); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	module, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, replacement := range module.Replace {
		if replacement.Old.Path == "example.org/contracts" && replacement.New.Path == contracts {
			found = true
		}
	}
	if !found {
		t.Fatalf("explicit local contract selection lost: %s", data)
	}
	if module.Require[1].Mod.Version != "v1.2.3" {
		t.Fatal("workspace changed the declared release requirement")
	}
}

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

func TestWriteRetainsPublishedGraphAndQuotesSourcePath(t *testing.T) {
	t.Setenv(WorkspaceEnv, "")
	root := filepath.Join(t.TempDir(), "source with spaces")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	const source = `module example.org/source

go 1.27.0

require example.org/contracts v1.2.3

replace example.org/contracts => ../unrelated-checkout
`
	const checksums = "example.org/contracts v1.2.3 h1:fixture\n"
	for name, content := range map[string]string{"go.mod": source, "go.sum": checksums} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	if err := Write(dir, "example.org/consumer", root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if consumer.Module.Mod.Path != "example.org/consumer" || consumer.Go.Version != "1.27.0" {
		t.Fatalf("consumer identity: %s", data)
	}
	if len(consumer.Replace) != 1 ||
		consumer.Replace[0].Old.Path != "example.org/source" ||
		consumer.Replace[0].New.Path != root {
		t.Fatalf("source-only replacement: %s", data)
	}
	requirements := map[string]string{}
	for _, requirement := range consumer.Require {
		requirements[requirement.Mod.Path] = requirement.Mod.Version
	}
	if len(requirements) != 2 ||
		requirements["example.org/source"] != "v0.0.0" ||
		requirements["example.org/contracts"] != "v1.2.3" {
		t.Fatalf("published graph: %s", data)
	}
	sum, err := os.ReadFile(filepath.Join(dir, "go.sum"))
	if err != nil || string(sum) != checksums {
		t.Fatalf("consumer checksums: %q, %v", sum, err)
	}
}

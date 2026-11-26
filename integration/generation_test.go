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

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/longxi-go/ioc/internal/testmodule"
)

func TestGeneratedConditionsComposeWithoutExecutingFactories(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := testmodule.Write(dir, "example.org/generatedconditions", root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/conditions/main.go.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	generate := exec.Command("go", "run", "./cmd/ioctl", "generate", "--dir", dir)
	generate.Dir = root
	output, err := generate.CombinedOutput()
	if err != nil {
		t.Fatalf("generate: %v\n%s", err, output)
	}
	if strings.Contains(string(output), "create user") || strings.Contains(string(output), "create fallback") {
		t.Fatalf("generation executed a factory: %s", output)
	}
	run := exec.Command("go", "run", ".")
	run.Dir = dir
	output, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("generated application: %v\n%s", err, output)
	}
	if string(output) != "create user\nuser\ncreate fallback\nfallback\n" {
		t.Fatalf("conditional contexts: %s", output)
	}
}

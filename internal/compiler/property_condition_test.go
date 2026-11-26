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
	"testing"
)

func TestGeneratedPropertyConditionUsesExactExistingValue(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "property-conditions/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "runtime_test.go"), fixtureSource(t, "property-conditions/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated property conditions: %v\n%s", err, output)
	}
}

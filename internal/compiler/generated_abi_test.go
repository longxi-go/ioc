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
	"strings"
	"testing"
)

func TestGeneratedBatchPinsGeneratorABI(t *testing.T) {
	source, err := emitPackage(&packageModel{
		Path: "example.org/app",
		Name: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if strings.Contains(text, "bean.SchemaVersion") || !strings.Contains(text, "Schema: 4,") {
		t.Fatal("generated batch must keep its generation-time ABI after a runtime upgrade")
	}
}

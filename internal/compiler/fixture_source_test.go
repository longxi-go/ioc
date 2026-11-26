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
	"os"
	"path/filepath"
	"testing"

	"github.com/longxi-go/gosr/pkg/codegen/render"
)

// fixtureSource keeps test programs readable as standalone Go files. The host
// package does not compile testdata; each consumer test decides how to load it.
func fixtureSource(t *testing.T, name string) string {
	t.Helper()
	return string(mustRead(t, filepath.Join("testdata", name)))
}

// fixtureTemplate uses the same strict renderer as generation only when a test
// genuinely needs substitutions, such as a cold module's unique import path.
func fixtureTemplate(t *testing.T, name string, view any) string {
	t.Helper()
	set, err := render.ParseFS(os.DirFS("testdata"), nil, name)
	if err != nil {
		t.Fatal(err)
	}
	data, err := set.Execute(filepath.Base(name), view)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

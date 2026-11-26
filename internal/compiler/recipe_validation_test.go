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
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidRecipeSemanticsFailBeforeOutput(t *testing.T) {
	cases := []struct {
		name, marker, fixture, want string
	}{
		{
			"context field",
			"",
			"recipe-validation/context-field.go.tmpl",
			"context.Context field injection",
		},
		{
			"empty property key",
			"// +ioc:conditional:on-property:name=\"\"",
			"recipe-validation/property.go.tmpl",
			"nonblank unique names",
		},
		{
			"missing property name",
			"// +ioc:conditional:on-property:havingValue=enabled",
			"recipe-validation/property.go.tmpl",
			"missing argument",
		},
		{
			"blank property key",
			"// +ioc:conditional:on-property:name=\" \"",
			"recipe-validation/property.go.tmpl",
			"nonblank unique names",
		},
		{
			"legacy property marker",
			"// +ioc:property=mode=user",
			"recipe-validation/property.go.tmpl",
			"unknown or misplaced marker",
		},
		{
			"lazy start",
			"// +ioc:lazy",
			"recipe-validation/lifecycle.go.tmpl",
			"Start requires eager singleton",
		},
		{
			"request start",
			"// +ioc:scope=request",
			"recipe-validation/lifecycle.go.tmpl",
			"Start requires eager singleton",
		},
		{
			"prototype start",
			"// +ioc:scope=prototype",
			"recipe-validation/lifecycle.go.tmpl",
			"Start requires eager singleton",
		},
		{
			"custom scoped start",
			"// +ioc:scope=session",
			"recipe-validation/lifecycle.go.tmpl",
			"Start requires eager singleton",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.ReplaceAll(fixtureSource(t, tc.fixture), "// RECIPE_SUPPLEMENT", tc.marker)
			dir := fixture(t, source)
			_, err := Run(context.Background(), Options{
				Dir: dir,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "main.go:") {
				t.Fatalf("want source diagnostic %q; got %v", tc.want, err)
			}
			if _, err := os.Stat(filepath.Join(dir, generatedFileName)); !os.IsNotExist(err) {
				t.Fatalf("invalid definition wrote output: %v", err)
			}
		})
	}
}

func TestValidRecipeSemanticsAllowEmptyPropertyValuesAndEagerStart(t *testing.T) {
	for _, name := range []string{"property", "lifecycle"} {
		t.Run(name, func(t *testing.T) {
			source := strings.ReplaceAll(fixtureSource(t, "recipe-validation/"+name+".go.tmpl"), "// RECIPE_SUPPLEMENT", "// +ioc:conditional:on-property:name=mode,havingValue=\"\"")
			dir := fixture(t, source)
			if _, err := Run(context.Background(), Options{
				Dir: dir,
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

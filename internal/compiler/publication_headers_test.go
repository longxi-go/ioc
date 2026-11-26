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
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/longxi-go/ioc/pkg/bean"
)

func TestPublicationVersionBoundaries(t *testing.T) {
	for _, test := range []struct {
		name   string
		schema int
		abi    int
	}{
		{
			name:   "schema",
			schema: -1,
			abi:    bean.SchemaVersion,
		},
		{
			name:   "recipe ABI",
			schema: schema,
			abi:    -1,
		},
		{
			name:   "previous publication schema",
			schema: 3,
			abi:    bean.SchemaVersion,
		},
		{
			name:   "previous recipe ABI",
			schema: schema,
			abi:    2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mod := module{
				Path: "example.org/header",
				Dir:  t.TempDir(),
			}
			data, err := json.Marshal(publication{
				Schema:    test.schema,
				RecipeABI: test.abi,
				Module:    mod.Path,
			})
			if err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(mod.Dir, "ioc/module.json"), string(data))
			_, _, err = readPublication(mod, target{}, nil)
			if err == nil || !strings.Contains(err.Error(), "schema, recipe ABI or module identity mismatch") {
				t.Fatalf("unsupported %s must fail before artifact loading: %v", test.name, err)
			}
			if !strings.Contains(err.Error(), "regenerate") {
				t.Fatalf("unsupported %s must instruct the library author to regenerate: %v", test.name, err)
			}
		})
	}
}

func TestLegacyPublicationFieldsRequireRegeneration(t *testing.T) {
	mod := module{
		Path: "example.org/legacy",
		Dir:  t.TempDir(),
	}
	write(t, filepath.Join(mod.Dir, "ioc/module.json"), `{
		"Schema": 3,
		"RecipeABI": 2,
		"Module": "example.org/legacy",
		"Recipes": [{"Auto": true, "Before": ["service"]}]
	}`)
	_, _, err := readPublication(mod, target{}, nil)
	if err == nil || !strings.Contains(err.Error(), "regenerate") {
		t.Fatalf("old fields must produce an actionable compatibility error: %v", err)
	}
}

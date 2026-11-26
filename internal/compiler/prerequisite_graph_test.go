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
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

func TestPrerequisiteGraphValidation(t *testing.T) {
	for _, scenario := range []string{"valid alias", "missing", "scope", "cycle", "duplicate alias", "open library", "conditional target"} {
		t.Run(scenario, func(t *testing.T) {
			a := publishedRecipe{
				Definition: gs.BeanDefinition{
					ID:        "a",
					Name:      "a",
					Scope:     gs.Singleton,
					DependsOn: []string{"alias"},
				},
			}
			b := publishedRecipe{
				Definition: gs.BeanDefinition{
					ID:      "b",
					Name:    "b",
					Aliases: []string{"alias"},
					Scope:   gs.Singleton,
				},
			}
			closed, valid := true, false
			switch scenario {
			case "valid alias":
				valid = true
			case "missing":
				a.Definition.DependsOn = []string{"absent"}
			case "scope":
				b.Definition.Scope = gs.Prototype
			case "cycle":
				b.Definition.Dependencies = []gs.Dependency{
					{
						Selector: gs.Selector{
							Name: "a",
						},
						Access:      gs.Direct,
						Cardinality: gs.One,
					},
				}
			case "duplicate alias":
				a.Definition.DependsOn = []string{"b", "alias"}
			case "open library":
				a.Definition.DependsOn = []string{"absent"}
				closed, valid = false, true
			case "conditional target":
				b.Conditional, valid = true, true
			}
			edges, err := validateGraphClosure([]publishedRecipe{a, b}, closed)
			if valid && err != nil ||
				!valid && err == nil {
				t.Fatalf("valid=%v got edges=%v err=%v", valid, edges, err)
			}
			if scenario == "valid alias" &&
				(len(edges) != 1 || edges[0].Kind != "depends-on" || edges[0].From != "a" || edges[0].To != "b") {
				t.Fatal(edges)
			}
		})
	}
}

func TestPrerequisiteGraphScopeMatrix(t *testing.T) {
	for _, from := range []gs.ScopeID{gs.Singleton, gs.Request, "tenant", gs.Prototype} {
		for _, to := range []gs.ScopeID{gs.Singleton, gs.Request, "tenant", gs.Prototype} {
			a := publishedRecipe{
				Definition: gs.BeanDefinition{
					ID:        "a",
					Name:      "a",
					Scope:     from,
					DependsOn: []string{"b"},
				},
			}
			b := publishedRecipe{
				Definition: gs.BeanDefinition{
					ID:    "b",
					Name:  "b",
					Scope: to,
				},
			}
			_, err := validateGraph([]publishedRecipe{a, b})
			allowed := to == gs.Singleton || from == to && from != gs.Prototype
			if allowed && err != nil ||
				!allowed && err == nil {
				t.Fatalf("%s -> %s: allowed=%v err=%v", from, to, allowed, err)
			}
		}
	}
}

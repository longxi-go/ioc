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

func TestGraphPrimarySelectionDoesNotInventSelfCycle(t *testing.T) {
	a := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:        "A",
			Name:      "A",
			Contracts: []gs.Contract{"R"},
			Scope:     gs.Singleton,
			Dependencies: []gs.Dependency{
				{
					Selector: gs.Selector{
						Contract: "R",
					},
					Access:      gs.Direct,
					Cardinality: gs.One,
				},
			},
		},
	}
	b := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:        "B",
			Name:      "B",
			Contracts: []gs.Contract{"R"},
			Scope:     gs.Singleton,
			Primary:   true,
		},
	}
	edges, err := validateGraph([]publishedRecipe{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 2 {
		t.Fatal("potential candidates were lost", edges)
	}
	b.Definition.Scope = gs.Request
	if _, err := validateGraph([]publishedRecipe{a, b}); err == nil {
		t.Fatal("selected short scope accepted")
	}
}

func TestGraphRejectsScopeEscapeThroughPlainPrototype(t *testing.T) {
	singleton := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:    "s",
			Name:  "s",
			Scope: gs.Singleton,
			Dependencies: []gs.Dependency{
				{
					Selector: gs.Selector{
						Contract: "P",
					},
					Access:      gs.Direct,
					Cardinality: gs.One,
				},
			},
		},
	}
	prototype := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:        "p",
			Name:      "p",
			Contracts: []gs.Contract{"P"},
			Scope:     gs.Prototype,
		},
	}
	if _, err := validateGraph([]publishedRecipe{singleton, prototype}); err != nil {
		t.Fatal("plain prototype retention is valid", err)
	}
	prototype.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Contract: "R",
			},
			Access:      gs.Direct,
			Cardinality: gs.One,
		},
	}
	request := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:        "r",
			Name:      "r",
			Contracts: []gs.Contract{"R"},
			Scope:     gs.Request,
		},
	}
	if _, err := validateGraph([]publishedRecipe{singleton, prototype, request}); err == nil {
		t.Fatal("short scope escaped through prototype")
	}
}

func TestGraphIncludesInitializationPrerequisiteEdges(t *testing.T) {
	a := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:        "A",
			Name:      "A",
			Scope:     gs.Singleton,
			DependsOn: []string{"B"},
		},
	}
	b := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:    "B",
			Name:  "B",
			Scope: gs.Singleton,
		},
	}
	edges, err := validateGraph([]publishedRecipe{a, b})
	if err != nil || len(edges) != 1 {
		t.Fatalf("order edge absent: %#v %v", edges, err)
	}
}

func TestPrerequisiteCannotActivateShorterScope(t *testing.T) {
	singleton := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:        "s",
			Name:      "s",
			Scope:     gs.Singleton,
			DependsOn: []string{"r"},
		},
	}
	request := publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:    "r",
			Name:  "r",
			Scope: gs.Request,
		},
	}
	if _, err := validateGraph([]publishedRecipe{singleton, request}); err == nil {
		t.Fatal("singleton cannot initialize an unowned request prerequisite")
	}
}

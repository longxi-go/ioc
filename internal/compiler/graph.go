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
	"sort"
)

// graphValidation separates potential report edges from definite creation and
// capture edges. Conditional candidates remain potential until runtime selection.
type graphValidation struct {
	recipes  []publishedRecipe
	closed   bool
	byID     map[string]publishedRecipe
	edges    []Edge
	direct   map[string][]string
	captures map[string][]string
}

func validateGraph(recipes []publishedRecipe) ([]Edge, error) {
	return validateGraphClosure(recipes, true)
}

func validateGraphClosure(recipes []publishedRecipe, closed bool) ([]Edge, error) {
	graph := graphValidation{
		recipes:  recipes,
		closed:   closed,
		byID:     make(map[string]publishedRecipe),
		captures: make(map[string][]string),
	}
	if err := graph.index(); err != nil {
		return nil, err
	}
	var err error
	graph.edges, graph.direct, err = prerequisiteGraph(recipes, closed)
	if err != nil {
		return nil, err
	}
	for _, recipe := range recipes {
		for _, dependency := range recipe.Definition.Dependencies {
			if err := graph.bind(recipe, dependency); err != nil {
				return nil, err
			}
		}
	}
	// Scope diagnostics precede cycle diagnostics, as in runtime-independent validation.
	if err := graph.validateCaptures(); err != nil {
		return nil, err
	}
	if err := graph.validateCycles(); err != nil {
		return nil, err
	}
	sort.Slice(graph.edges, func(i, j int) bool {
		return edgeLess(graph.edges[i], graph.edges[j])
	})
	return graph.edges, nil
}

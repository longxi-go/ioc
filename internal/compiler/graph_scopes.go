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
	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const transitiveScopeEscapeMessage = "singleton captures a shorter owned scope through a direct dependency"

type captureTraversal struct {
	graph *graphValidation
	root  gs.BeanDefinition
	seen  map[string]bool
}

func shorterOwnedScope(scope gs.ScopeID) bool {
	return scope != gs.Singleton && scope != gs.Prototype
}

func (g *graphValidation) validateCaptures() error {
	for _, root := range g.recipes {
		if root.Definition.Scope != gs.Singleton || root.Conditional {
			continue
		}
		traversal := captureTraversal{
			graph: g,
			root:  root.Definition,
			seen:  make(map[string]bool),
		}
		if err := traversal.inspect(root.Definition.ID, []string{root.Definition.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (v *captureTraversal) inspect(id string, path []string) error {
	if v.seen[id] {
		return nil
	}
	v.seen[id] = true
	for _, next := range v.graph.captures[id] {
		target := v.graph.byID[next]
		if shorterOwnedScope(target.Definition.Scope) {
			return &gs.Error{
				Code:    gs.ScopeViolation,
				Bean:    v.root.ID,
				Source:  v.root.Source,
				Path:    append(append([]string{}, path...), next),
				Message: transitiveScopeEscapeMessage,
			}
		}
		if target.Definition.Scope == gs.Prototype {
			if err := v.inspect(next, append(append([]string{}, path...), next)); err != nil {
				return err
			}
		}
	}
	return nil
}

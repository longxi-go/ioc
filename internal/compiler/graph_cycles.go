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

const directDependencyCycleMessage = "direct dependency cycle"

type dependencyTraversal struct {
	graph *graphValidation
	state map[string]visitState
	path  []string
}

func (g *graphValidation) validateCycles() error {
	traversal := dependencyTraversal{
		graph: g,
		state: make(map[string]visitState),
	}
	for _, r := range g.recipes {
		if err := traversal.visit(r.Definition.ID); err != nil {
			return err
		}
	}
	return nil
}

func (v *dependencyTraversal) visit(id string) error {
	if v.state[id] == visiting {
		return &gs.Error{
			Code:    gs.DependencyCycle,
			Bean:    id,
			Source:  v.graph.byID[id].Definition.Source,
			Path:    append(append([]string{}, v.path...), id),
			Message: directDependencyCycleMessage,
		}
	}
	if v.state[id] == visited {
		return nil
	}
	v.state[id] = visiting
	v.path = append(v.path, id)
	for _, next := range v.graph.direct[id] {
		if err := v.visit(next); err != nil {
			return err
		}
	}
	v.path = v.path[:len(v.path)-1]
	v.state[id] = visited
	return nil
}

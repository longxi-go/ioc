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
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	missingInitializationPrerequisiteMessage     = "missing initialization prerequisite "
	unsupportedInitializationOwnershipForMessage = "unsupported initialization ownership for "
)

// prerequisiteGraph distinguishes portable potential edges from edges whose
// activation is certain. Open libraries may receive missing targets later.
func prerequisiteGraph(recipes []publishedRecipe, closed bool) ([]Edge, map[string][]string, error) {
	var edges []Edge
	direct := make(map[string][]string, len(recipes))
	for _, r := range recipes {
		d := r.Definition
		resolved := make(map[string]bool, len(d.DependsOn))
		for _, name := range d.DependsOn {
			var matches []publishedRecipe
			for _, candidate := range recipes {
				cd := candidate.Definition
				if cd.Name == name || slices.Contains(cd.Aliases, name) {
					matches = append(matches, candidate)
					edges = append(edges, Edge{
						From:   d.ID,
						To:     cd.ID,
						Kind:   PrerequisiteEdge,
						Access: gs.Direct,
						Source: d.Source,
					})
				}
			}
			if closed && !r.Conditional && len(matches) == 0 {
				return nil, nil, prerequisiteFailure(gs.MissingBean, d, missingInitializationPrerequisiteMessage+name)
			}
			if r.Conditional || len(matches) != 1 || matches[0].Conditional {
				continue
			}
			target := matches[0].Definition
			if target.ID == d.ID || resolved[target.ID] {
				return nil, nil, prerequisiteFailure(gs.InvalidDefinition, d, bean.InvalidPrerequisiteMessage+name)
			}
			resolved[target.ID] = true
			if target.Scope != gs.Singleton && (target.Scope != d.Scope || target.Scope == gs.Prototype) {
				return nil, nil, prerequisiteFailure(gs.ScopeViolation, d, unsupportedInitializationOwnershipForMessage+name)
			}
			direct[d.ID] = append(direct[d.ID], target.ID)
		}
	}
	return edges, direct, nil
}

func prerequisiteFailure(code gs.Code, d gs.BeanDefinition, message string) error {
	return &gs.Error{
		Code:    code,
		Bean:    d.ID,
		Source:  d.Source,
		Message: message,
	}
}

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
	"fmt"
	"sort"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	ambiguousDependencyForFormat       = "ambiguous dependency %s for %s"
	missingDependencyForFormat         = "missing dependency %s for %s"
	scopeEscapeSingletonCapturesFormat = "scope escape: singleton %s captures %s"
)

type dependencyCandidates struct {
	matches       []publishedRecipe
	unconditional int
}

func (g *graphValidation) candidates(selector gs.Selector) dependencyCandidates {
	var result dependencyCandidates
	for _, candidate := range g.recipes {
		if !candidate.Definition.Matches(selector) {
			continue
		}
		result.matches = append(result.matches, candidate)
		if !candidate.Conditional {
			result.unconditional++
		}
	}
	sort.Slice(result.matches, func(i, j int) bool {
		return result.matches[i].Definition.ID < result.matches[j].Definition.ID
	})
	return result
}

func (c dependencyCandidates) allUnconditional() bool {
	return len(c.matches) == c.unconditional
}

func (c dependencyCandidates) primaryCount() int {
	count := 0
	for _, match := range c.matches {
		if match.Definition.Primary {
			count++
		}
	}
	return count
}

func (g *graphValidation) bind(owner publishedRecipe, dep gs.Dependency) error {
	candidates := g.candidates(dep.Selector)
	matches := candidates.matches
	definiteOwner := g.closed && !owner.Conditional
	if definiteOwner && len(matches) == 0 && dep.Cardinality == gs.One {
		return fmt.Errorf(missingDependencyForFormat, dep.Selector.Contract, owner.Definition.ID)
	}
	ambiguous := len(matches) > 1 && dep.Cardinality != gs.Many
	if definiteOwner &&
		ambiguous &&
		candidates.allUnconditional() &&
		candidates.primaryCount() != 1 {
		return fmt.Errorf(ambiguousDependencyForFormat, dep.Selector.Contract, owner.Definition.ID)
	}
	certain := (g.closed || dep.Selector.Name != "") &&
		candidates.allUnconditional() && !owner.Conditional
	for _, match := range matches {
		g.edges = append(g.edges, Edge{
			From:   owner.Definition.ID,
			To:     match.Definition.ID,
			Access: dep.Access,
			Kind:   DependencyEdge,
			Source: dep.Source,
		})
		chosen := !ambiguous || match.Definition.Primary
		if dep.Access != gs.Direct || !certain || !chosen {
			continue
		}
		g.direct[owner.Definition.ID] = append(g.direct[owner.Definition.ID], match.Definition.ID)
		g.captures[owner.Definition.ID] = append(g.captures[owner.Definition.ID], match.Definition.ID)
		if owner.Definition.Scope == gs.Singleton && shorterOwnedScope(match.Definition.Scope) {
			return fmt.Errorf(scopeEscapeSingletonCapturesFormat, owner.Definition.ID, match.Definition.ID)
		}
	}
	return nil
}

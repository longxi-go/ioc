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

package container

import (
	"cmp"
	"fmt"
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	activeNameOrAliasConflictMessage = "active name or alias conflict"

	cyclicCreationDependenciesMessage        = "cyclic creation dependencies"
	cyclicCreationOrCandidateOrderingMessage = "cyclic creation or candidate ordering"
	dependencyPortFormat                     = "dependency port %d"
	directScopeEscapeMessage                 = "direct dependency captures a shorter or unrelated lifetime"
	duplicateActiveIdentityMessage           = "duplicate active identity"

	multipleCandidatesWithoutAUniquePrimaryMessage  = "multiple candidates without a unique primary"
	multiplePrimaryCandidatesMessage                = "multiple primary candidates"
	namedBeanHasIncompatibleContractMessage         = "named bean has incompatible contract"
	noCandidateForSelectorMessage                   = "no candidate for selector"
	resourcePrototypeRequiresAnExplicitLeaseMessage = "resource prototype requires an explicit lease"
	transitiveDependencyEscapesOwnerLifetimeMessage = "transitive dependency escapes owner lifetime"
	unsupportedScopeMessage                         = "unsupported scope"
)

type bound struct {
	recipe        bean.Registration
	ports         [][]*bound
	prerequisites []*bound
	singleton     *slot
}

func failure(code gs.Code, id, message string, cause error) error {
	return &gs.Error{
		Code:    code,
		Bean:    id,
		Message: message,
		Cause:   cause,
	}
}

func topological(edges []map[int]bool) ([]int, error) {
	indegree := make([]int, len(edges))
	for _, e := range edges {
		for j := range e {
			indegree[j]++
		}
	}
	out := make([]int, 0, len(edges))
	used := make([]bool, len(edges))
	for len(out) < len(edges) {
		found := -1
		for i, n := range indegree {
			if n == 0 && !used[i] {
				found = i
				break
			}
		}
		if found < 0 {
			return nil, failure(gs.DependencyCycle, "", cyclicCreationOrCandidateOrderingMessage, nil)
		}
		used[found] = true
		out = append(out, found)
		for j := range edges[found] {
			indegree[j]--
		}
	}
	return out, nil
}

func choose(bs []*bound, s gs.Selector, many, optional bool) ([]*bound, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	var matches []*bound
	var named *bound
	for _, b := range bs {
		d := b.recipe.Definition
		if s.Name != "" && (d.Name == s.Name || slices.Contains(d.Aliases, s.Name)) {
			named = b
		}
		if d.Matches(s) {
			matches = append(matches, b)
		}
	}
	if named != nil && s.Contract != "" && !slices.Contains(named.recipe.Definition.Contracts, s.Contract) {
		return nil, failure(gs.ContractMismatch, named.recipe.Definition.ID, namedBeanHasIncompatibleContractMessage, nil)
	}
	if many {
		slices.SortFunc(matches, compareCollectionMembers)
		return matches, nil
	}
	if len(matches) == 0 {
		if optional {
			return nil, nil
		}
		return nil, failure(gs.MissingBean, "", noCandidateForSelectorMessage, nil)
	}
	if len(matches) == 1 {
		return matches, nil
	}
	var primary *bound
	for _, b := range matches {
		if b.recipe.Definition.Primary {
			if primary != nil {
				return nil, failure(gs.AmbiguousBean, "", multiplePrimaryCandidatesMessage, nil)
			}
			primary = b
		}
	}
	if primary != nil {
		return []*bound{primary}, nil
	}
	return nil, failure(gs.AmbiguousBean, "", multipleCandidatesWithoutAUniquePrimaryMessage, nil)
}

func compareCollectionMembers(a, b *bound) int {
	if n := cmp.Compare(a.recipe.Definition.Order, b.recipe.Definition.Order); n != 0 {
		return n
	}
	return cmp.Compare(a.recipe.Definition.ID, b.recipe.Definition.ID)
}

func bind(rs []bean.Registration, opts Options) ([]*bound, []*bound, error) {
	active, err := selectRecipes(rs, opts.Snapshot)
	if err != nil {
		return nil, nil, err
	}
	// Independent creation ties use identity, never collection Order.
	ordered(active)
	bs := make([]*bound, len(active))
	names := map[string]bool{}
	ids := map[string]int{}
	for i, r := range active {
		d := r.Definition
		if err = d.Validate(); err != nil {
			return nil, nil, err
		}
		if r.Create == nil {
			return nil, nil, failure(gs.InvalidDefinition, d.ID, bean.MissingFactoryMessage, nil)
		}
		if _, exists := ids[d.ID]; exists {
			return nil, nil, failure(gs.DuplicateDefinition, d.ID, duplicateActiveIdentityMessage, nil)
		}
		ids[d.ID] = i
		for _, name := range append([]string{d.Name}, d.Aliases...) {
			if names[name] {
				return nil, nil, failure(gs.DuplicateDefinition, d.ID, activeNameOrAliasConflictMessage, nil)
			}
			names[name] = true
		}
		if d.Scope != gs.Singleton &&
			d.Scope != gs.Prototype &&
			d.Scope != gs.Request &&
			!slices.Contains(opts.Scopes, d.Scope) {
			return nil, nil, failure(gs.ScopeViolation, d.ID, unsupportedScopeMessage, nil)
		}
		if r.Start != nil && (d.Scope != gs.Singleton || d.Lazy) {
			return nil, nil, failure(gs.InvalidDefinition, d.ID, bean.EagerSingletonStartMessage, nil)
		}
		bs[i] = &bound{
			recipe:    r,
			singleton: &slot{},
		}
	}
	edges := make([]map[int]bool, len(bs))
	if err := bindPrerequisites(bs); err != nil {
		return nil, nil, err
	}
	for i := range edges {
		edges[i] = map[int]bool{}
	}
	for i, b := range bs {
		d := b.recipe.Definition
		for _, target := range b.prerequisites {
			edges[ids[target.recipe.Definition.ID]][i] = true
		}
		b.ports = make([][]*bound, len(d.Dependencies))
		for port, dep := range d.Dependencies {
			targets, err := choose(bs, dep.Selector, dep.Cardinality == gs.Many, dep.Cardinality == gs.ZeroOrOne)
			if err != nil {
				return nil, nil, &gs.Error{
					Code:    codeOf(err),
					Bean:    d.ID,
					Source:  dep.Source,
					Message: fmt.Sprintf(dependencyPortFormat, port),
					Cause:   err,
				}
			}
			b.ports[port] = targets
			if dep.Access == gs.Deferred {
				continue
			}
			for _, target := range targets {
				td := target.recipe.Definition
				if lifetimeEscape(d.Scope, td.Scope) {
					return nil, nil, failure(gs.ScopeViolation, d.ID, directScopeEscapeMessage, nil)
				}
				if td.Scope == gs.Prototype && (target.recipe.Stop != nil || target.recipe.Destroy != nil) {
					return nil, nil, failure(gs.ScopeViolation, d.ID, resourcePrototypeRequiresAnExplicitLeaseMessage, nil)
				}
				edges[ids[td.ID]][i] = true
			}
		}
	}
	if err := validateTransitiveScopes(bs); err != nil {
		return nil, nil, err
	}
	order, err := topological(edges)
	if err != nil {
		return nil, nil, graphFailure(active, edges)
	}
	sorted := make([]*bound, len(order))
	for i, j := range order {
		sorted[i] = bs[j]
	}
	return bs, sorted, nil
}

func codeOf(err error) gs.Code {
	if e, ok := err.(*gs.Error); ok {
		return e.Code
	}
	return gs.CreationFailed
}

func graphFailure(rs []bean.Registration, edges []map[int]bool) *gs.Error {
	marks := make([]visitState, len(rs))
	var stack, cycle []int
	var visit candidateVisitFunc
	visit = func(i int) bool {
		if marks[i] == visiting {
			at := slices.Index(stack, i)
			cycle = append(slices.Clone(stack[at:]), i)
			return true
		}
		if marks[i] == visited {
			return false
		}
		marks[i] = visiting
		stack = append(stack, i)
		for j := range edges[i] {
			if visit(j) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		marks[i] = visited
		return false
	}
	for i := range rs {
		if visit(i) {
			break
		}
	}
	err := &gs.Error{
		Code:    gs.DependencyCycle,
		Message: cyclicCreationDependenciesMessage,
	}
	for _, i := range cycle {
		err.Path = append(err.Path, rs[i].Definition.ID)
	}
	if len(cycle) > 0 {
		d := rs[cycle[0]].Definition
		err.Bean = d.ID
		err.Source = d.Source
	}
	for k := 1; k < len(cycle); k++ {
		dependency := rs[cycle[k-1]].Definition
		consumer := rs[cycle[k]].Definition
		for _, dep := range consumer.Dependencies {
			if dep.Access == gs.Direct && dependency.Matches(dep.Selector) {
				err.Source = dep.Source
				err.Bean = consumer.ID
				return err
			}
		}
	}
	return err
}

func lifetimeEscape(from, to gs.ScopeID) bool {
	if from == gs.Singleton {
		return to != gs.Singleton && to != gs.Prototype
	}
	return from != gs.Prototype && to != gs.Singleton && to != gs.Prototype && to != from
}

func validateTransitiveScopes(bs []*bound) error {
	for _, root := range bs {
		if root.recipe.Definition.Scope == gs.Prototype {
			continue
		}
		seen := map[*bound]bool{}
		var visit dependencyInspectFunc
		visit = func(b *bound, path []string) error {
			if seen[b] {
				return nil
			}
			seen[b] = true
			for i, targets := range b.ports {
				dep := b.recipe.Definition.Dependencies[i]
				if dep.Access != gs.Direct {
					continue
				}
				for _, target := range targets {
					next := append(slices.Clone(path), target.recipe.Definition.ID)
					if lifetimeEscape(root.recipe.Definition.Scope, target.recipe.Definition.Scope) {
						return &gs.Error{
							Code:    gs.ScopeViolation,
							Bean:    root.recipe.Definition.ID,
							Source:  dep.Source,
							Path:    next,
							Message: transitiveDependencyEscapesOwnerLifetimeMessage,
						}
					}
					if target.recipe.Definition.Scope == gs.Prototype {
						if err := visit(target, next); err != nil {
							return err
						}
					}
				}
			}
			return nil
		}
		if err := visit(root, []string{root.recipe.Definition.ID}); err != nil {
			return err
		}
	}
	return nil
}

func recipeFailure(r bean.Registration, code gs.Code, message string, cause error) error {
	return &gs.Error{
		Code:    code,
		Bean:    r.Definition.ID,
		Source:  r.Definition.Source,
		Message: message,
		Cause:   cause,
	}
}

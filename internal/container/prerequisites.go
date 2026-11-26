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
	"context"
	"fmt"
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	initializationPrerequisiteMessage              = "initialization prerequisite "
	missingActiveInitializationPrerequisiteMessage = "missing active initialization prerequisite "

	unsupportedInitializationOwnershipDependsOnFormat = "unsupported initialization ownership: %s depends on %s"
)

// bindPrerequisites resolves active names only. Prerequisites are initialization
// edges, so they neither consume ports nor apply type/primary selection.
func bindPrerequisites(bs []*bound) error {
	byName := make(map[string]*bound, len(bs))
	for _, b := range bs {
		d := b.recipe.Definition
		byName[d.Name] = b
		for _, alias := range d.Aliases {
			byName[alias] = b
		}
	}
	for _, b := range bs {
		d := b.recipe.Definition
		for _, name := range d.DependsOn {
			target := byName[name]
			if target == nil {
				return recipeFailure(b.recipe, gs.MissingBean, missingActiveInitializationPrerequisiteMessage+name, nil)
			}
			if target == b || slices.Contains(b.prerequisites, target) {
				return recipeFailure(b.recipe, gs.InvalidDefinition, bean.InvalidPrerequisiteMessage+name, nil)
			}
			if err := validatePrerequisiteScope(d.Scope, target.recipe.Definition.Scope); err != nil {
				return recipeFailure(b.recipe, gs.ScopeViolation, initializationPrerequisiteMessage+name, err)
			}
			b.prerequisites = append(b.prerequisites, target)
		}
	}
	return nil
}

func validatePrerequisiteScope(from, to gs.ScopeID) error {
	if to == gs.Singleton || from == to && from != gs.Prototype {
		return nil
	}
	return fmt.Errorf(unsupportedInitializationOwnershipDependsOnFormat, from, to)
}

// initializePrerequisites uses the same tracked resolution path as injection.
// A child task must not overwrite its parent's current bean or wait-graph node.
func (e *Engine) initializePrerequisites(ctx context.Context, b *bound, o *owner) error {
	for _, target := range b.prerequisites {
		call, child, finish, err := e.begin(ctx)
		if err != nil {
			return err
		}
		if o != nil && target.recipe.Definition.Scope == b.recipe.Definition.Scope {
			// Keep the dependent's exact scoped capability, including nested
			// owners of the same kind; no scope or lease is created here.
			previous, _ := call.Value(scopeKey{}).(*scopeChain)
			call = context.WithValue(call, scopeKey{}, &scopeChain{
				owner:    o,
				previous: previous,
			})
		}
		_, err = e.get(call, target, resolution{
			task: child,
		})
		finish()
		if err != nil {
			return recipeFailure(b.recipe, codeOf(err), initializationPrerequisiteMessage+target.recipe.Definition.Name, err)
		}
	}
	return nil
}

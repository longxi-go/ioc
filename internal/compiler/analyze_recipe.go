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
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"golang.org/x/tools/go/packages"
)

const (
	duplicateExposedContractFormat                   = "duplicate exposed contract %s"
	exposeFormat                                     = "expose %q: %w"
	exposedTypeLabel                                 = "exposed type"
	hookMustAcceptContextContextAndReturnErrorFormat = "%s hook must accept context.Context and return error"
	resultDoesNotImplementExposedFormat              = "result %s does not implement exposed %s"
)

func completeRecipe(pkg *packages.Package, r *recipe, a annotation, pos token.Pos) error {
	r.ListenerType = a.flag(markerListener)
	name := r.Function
	if name == "" {
		name = r.Component
	}
	d := &r.Definition
	d.ID = pkg.PkgPath + definitionMemberSeparator + name
	d.Name = a.one(markerName)
	if d.Name == "" {
		d.Name = name
	}
	d.Aliases = a.strings(markerAlias)
	d.Qualifiers = a.strings(markerQualifier)
	d.Scope = gs.ScopeID(a.one(markerScope))
	if d.Scope == "" {
		d.Scope = gs.Singleton
	}
	d.Primary = a.flag(markerPrimary)
	d.Lazy = a.flag(markerLazy)
	d.Source = source(pkg, pos)
	if values := a[markerOrder]; len(values) > 0 {
		d.Order = values[0].(int)
	}
	r.Exposed = []types.Type{r.Result}
	d.Contracts = []gs.Contract{contract(r.Result)}
	for _, text := range a.strings(markerExpose) {
		exposed, err := resolveTypeReference(pkg, pos, text, exposedTypeLabel)
		if err != nil {
			return at(pkg, pos, fmt.Errorf(exposeFormat, text, err))
		}
		if !types.AssignableTo(r.Result, exposed) {
			return at(pkg, pos, fmt.Errorf(resultDoesNotImplementExposedFormat, r.Result, text))
		}
		c := contract(exposed)
		if slices.Contains(d.Contracts, c) {
			return at(pkg, pos, fmt.Errorf(duplicateExposedContractFormat, c))
		}
		r.Exposed = append(r.Exposed, exposed)
		d.Contracts = append(d.Contracts, c)
	}
	prerequisites, err := compilePrerequisiteNames(a)
	if err != nil {
		return at(pkg, pos, err)
	}
	d.DependsOn = prerequisites
	conditions, err := compileConditions(pkg, pos, r.Result, a)
	if err != nil {
		return at(pkg, pos, err)
	}
	r.Conditions = append(r.Conditions, conditions...)
	for _, hook := range []lifecycleHook{initHook, startHook, stopHook, destroyHook} {
		selection := types.NewMethodSet(r.Result).Lookup(nil, string(hook))
		if selection == nil {
			continue
		}
		sig := selection.Obj().Type().(*types.Signature)
		if sig.Params().Len() != 1 || !isContext(sig.Params().At(0).Type()) ||
			sig.Results().Len() != 1 ||
			!types.Identical(sig.Results().At(0).Type(), types.Universe.Lookup(errorTypeName).Type()) {
			return at(pkg, pos, fmt.Errorf(hookMustAcceptContextContextAndReturnErrorFormat, hook))
		}
		r.Hooks = append(r.Hooks, hook)
		if hook == startHook && (d.Scope != gs.Singleton || d.Lazy) {
			return at(pkg, pos, errors.New(bean.EagerSingletonStartMessage))
		}
	}
	if err := d.Validate(); err != nil {
		return at(pkg, pos, err)
	}
	return nil
}

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
	"go/ast"
	"go/types"

	"github.com/longxi-go/ioc/internal/httpcompiler"
)

const (
	missingBeanMarkerMessage                          = "supplement requires +ioc:bean"
	componentMarkerRequiresAStructMessage             = "component marker requires a struct"
	factoryResultMustBeTOrTErrorMessage               = "factory result must be T or (T,error)"
	listenOnlyAcceptsAnOptionalOrderSupplementMessage = "listen only accepts an optional order supplement"
	methodFactoriesAreUnsupportedMessage              = "method factories are unsupported"
	openGenericFactoriesAreUnsupportedMessage         = "open generic factories are unsupported"
	variadicFactoriesAreUnsupportedMessage            = "variadic factories are unsupported"
)

func (analysis *packageAnalysis) function(d *ast.FuncDecl) error {
	pkg := analysis.pkg
	a, err := readMarkers(pkg.Fset, d.Doc, declarationTarget)
	if err != nil {
		return at(pkg, d.Pos(), err)
	}
	if a.flag(httpcompiler.Controller) {
		return at(pkg, d.Pos(), errors.New(componentMarkerRequiresAStructMessage))
	}
	if hasHTTP(a) {
		return validateHTTPFunction(pkg, d, a)
	}
	if a.flag(markerComponent) {
		return at(pkg, d.Pos(), errors.New(componentMarkerRequiresAStructMessage))
	}
	if a.flag(markerListen) {
		if d.Recv == nil {
			return at(pkg, d.Pos(), errors.New(listenRequiresAReceiverMethodMessage))
		}
		for key := range a {
			if key != markerListen && key != markerOrder {
				return at(pkg, d.Pos(), errors.New(listenOnlyAcceptsAnOptionalOrderSupplementMessage))
			}
		}
		return nil
	}
	if !a.flag(markerBean) {
		if len(a) > 0 {
			return at(pkg, d.Pos(), errors.New(missingBeanMarkerMessage))
		}
		return nil
	}
	if d.Recv != nil {
		return at(pkg, d.Pos(), errors.New(methodFactoriesAreUnsupportedMessage))
	}
	obj := pkg.TypesInfo.Defs[d.Name]
	sig := obj.Type().(*types.Signature)
	if sig.TypeParams() != nil && sig.TypeParams().Len() > 0 {
		return at(pkg, d.Pos(), errors.New(openGenericFactoriesAreUnsupportedMessage))
	}

	if sig.Variadic() {
		return at(pkg, d.Pos(), errors.New(variadicFactoriesAreUnsupportedMessage))
	}
	if sig.Results().Len() < 1 || sig.Results().Len() > 2 ||
		(sig.Results().Len() == 2 &&
			!types.Identical(sig.Results().At(1).Type(), types.Universe.Lookup(errorTypeName).Type())) {
		return at(pkg, d.Pos(), errors.New(factoryResultMustBeTOrTErrorMessage))
	}
	r := &recipe{
		Package:  pkg.PkgPath,
		Function: d.Name.Name,
		Result:   sig.Results().At(0).Type(),
		Error:    sig.Results().Len() == 2,
	}
	if err := analysis.parameters(r, parameterAnalysis{
		signature: sig,
		markers:   a,
		position:  d.Pos(),
	}); err != nil {
		return err
	}

	if err := completeRecipe(pkg, r, a, d.Pos()); err != nil {
		return err
	}
	analysis.model.Recipes = append(analysis.model.Recipes, r)

	return nil
}

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

	sourceannotation "github.com/longxi-go/gosr/tools/pkg/annotation"
)

const (
	missingComponentMarkerMessage                  = "supplement requires +ioc:component"
	beanMarkerRequiresAFunctionMessage             = "bean marker requires a function"
	componentRequiresAStructMessage                = "component requires a struct"
	openGenericComponentsAreUnsupportedMessage     = "open generic components are unsupported"
	paramMarkersRequireAConstructorFunctionMessage = "param markers require a constructor function"
)

func (analysis *packageAnalysis) types(d *ast.GenDecl) error {
	pkg := analysis.pkg
	for _, spec := range d.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		group := sourceannotation.TypeComments(d, ts)
		a, err := readMarkers(pkg.Fset, group, declarationTarget)
		if err != nil {
			return at(pkg, ts.Pos(), err)
		}
		if a.flag(markerBean) {
			return at(pkg, ts.Pos(), errors.New(beanMarkerRequiresAFunctionMessage))
		}
		if a.flag(markerParam) {
			return at(pkg, ts.Pos(), errors.New(paramMarkersRequireAConstructorFunctionMessage))
		}
		if a.flag(markerListen) {
			return at(pkg, ts.Pos(), errors.New(listenRequiresAReceiverMethodMessage))
		}
		if !a.flag(markerComponent) {
			if len(a) > 0 {
				return at(pkg, ts.Pos(), errors.New(missingComponentMarkerMessage))
			}
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return at(pkg, ts.Pos(), errors.New(componentRequiresAStructMessage))
		}
		obj := pkg.TypesInfo.Defs[ts.Name]
		named, ok := obj.Type().(*types.Named)
		if !ok || named.TypeParams().Len() > 0 {
			return at(pkg, ts.Pos(), errors.New(openGenericComponentsAreUnsupportedMessage))
		}

		r := &recipe{
			Package:      pkg.PkgPath,
			Component:    ts.Name.Name,
			Result:       types.NewPointer(named),
			ListenerType: a.flag(markerListener),
		}
		if err := analysis.fields(r, st); err != nil {
			return err
		}

		if err := completeRecipe(pkg, r, a, ts.Pos()); err != nil {
			return err
		}
		analysis.model.Recipes = append(analysis.model.Recipes, r)
	}
	return nil
}

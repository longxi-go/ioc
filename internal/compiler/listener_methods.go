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

	"golang.org/x/tools/go/packages"
)

func annotatedListenerMethods(pkg *packages.Package) ([]*annotatedMethod, error) {
	var methods []*annotatedMethod
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			method, ok := declaration.(*ast.FuncDecl)
			if !ok || method.Recv == nil {
				continue
			}
			a, err := readMarkers(pkg.Fset, method.Doc, declarationTarget)
			if err != nil || !a.flag(markerListen) {
				continue
			}
			object := pkg.TypesInfo.Defs[method.Name].(*types.Func)
			sig := object.Type().(*types.Signature)
			receiver := types.Unalias(sig.Recv().Type())
			if pointer, ok := receiver.(*types.Pointer); ok {
				receiver = types.Unalias(pointer.Elem())
			}
			if named, ok := receiver.(*types.Named); ok && named.TypeParams().Len() > 0 {
				return nil, at(pkg, method.Pos(), errors.New(openGenericListenerReceiversAreUnsupportedMessage))
			}
			if _, err := listenerEvent(sig); err != nil {
				return nil, at(pkg, method.Pos(), err)
			}
			annotation := &annotatedMethod{
				object: object,
				pos:    method.Pos(),
			}
			if values := a[markerOrder]; len(values) > 0 {
				order := values[0].(int)
				annotation.order = &order
			}
			methods = append(methods, annotation)
		}
	}
	return methods, nil
}

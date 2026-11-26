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
)

const (
	missingInjectMarkerMessage                      = "field supplement requires +ioc:inject"
	contextFieldInjectionMessage                    = "context.Context field injection is unsupported; receive the operation context in constructor or lifecycle parameters"
	injectionIntoEmbeddedFieldsIsUnsupportedMessage = "injection into embedded fields is unsupported"
)

func (analysis *packageAnalysis) fields(r *recipe, st *ast.StructType) error {
	pkg := analysis.pkg
	for _, field := range st.Fields.List {
		fa, err := readMarkers(pkg.Fset, field.Doc, fieldTarget)
		if err != nil {
			return at(pkg, field.Pos(), err)
		}
		if !fa.flag(markerInject) {
			if len(fa) > 0 {
				return at(pkg, field.Pos(), errors.New(missingInjectMarkerMessage))
			}
			continue
		}
		if len(field.Names) == 0 {
			return at(pkg, field.Pos(), errors.New(injectionIntoEmbeddedFieldsIsUnsupportedMessage))
		}
		for _, name := range field.Names {
			fieldType := pkg.TypesInfo.Defs[name].Type()
			if isContext(fieldType) {
				return at(pkg, field.Pos(), errors.New(contextFieldInjectionMessage))
			}
			if err := r.addArg(fieldType, name.Name, fa, source(pkg, field.Pos())); err != nil {
				return at(pkg, field.Pos(), err)
			}
		}
	}
	return nil
}

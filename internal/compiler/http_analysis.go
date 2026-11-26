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
	"go/token"
	"go/types"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/internal/httpcompiler"
	http "github.com/longxi-go/ioc/pkg/codegen/http"
	"golang.org/x/tools/go/packages"
)

func (analysis *packageAnalysis) controller(r *recipe, a annotation, pos token.Pos) error {
	if r.Definition.Scope != gs.Singleton && r.Definition.Scope != gs.Request {
		return at(analysis.pkg, pos, errors.New(httpControllerScopeMessage))
	}
	if hasHTTP(a) {
		marker, err := httpMarker(a)
		if err != nil {
			return at(analysis.pkg, pos, err)
		}
		if marker.Method != "" {
			return at(analysis.pkg, pos, errors.New(httpBaseMethodMessage))
		}
		if analysis.controllerBases == nil {
			analysis.controllerBases = map[string]string{}
		}
		analysis.controllerBases["*"+r.Component] = marker.Path
	}
	analysis.model.HTTP.Controllers = append(analysis.model.HTTP.Controllers, http.Controller{
		Definition: r.Definition.Clone(),
		Type:       "*" + r.Component,
	})
	return nil
}

func validateHTTPFunction(pkg *packages.Package, d *ast.FuncDecl, a annotation) error {
	marker, err := httpMarker(a)
	if err != nil {
		return at(pkg, d.Pos(), err)
	}
	if err := httpcompiler.Method(marker.Method); err != nil {
		return at(pkg, d.Pos(), err)
	}
	for key := range a {
		if key != httpcompiler.Marker {
			if _, ok := httpcompiler.Shortcuts()[key]; !ok {
				return at(pkg, d.Pos(), errors.New(httpEndpointSupplementsMessage))
			}
		}
	}
	return nil
}

func (analysis *packageAnalysis) attachHTTP() error {
	pkg, model := analysis.pkg, analysis.model
	model.HTTP.Path = model.Path
	model.HTTP.Name = model.Name
	for _, f := range pkg.Syntax {
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			a, err := readMarkers(pkg.Fset, d.Doc, declarationTarget)
			if err != nil {
				return err
			}
			if !hasHTTP(a) {
				continue
			}
			m, err := httpMarker(a)
			if err != nil {
				return err
			}
			sig := pkg.TypesInfo.Defs[d.Name].Type().(*types.Signature)
			route := http.Route{
				Method:    m.Method,
				Path:      m.Path,
				Handler:   d.Name.Name,
				Signature: sig,
				Source:    source(pkg, d.Pos()),
			}
			if sig.Recv() == nil {
				model.HTTP.Handlers = append(model.HTTP.Handlers, route)
				continue
			}
			recv := sig.Recv().Type()
			if pointer, ok := recv.(*types.Pointer); ok {
				recv = pointer.Elem()
			}
			named, ok := recv.(*types.Named)
			if !ok {
				return at(pkg, d.Pos(), errors.New(httpReceiverMessage))
			}
			found := false
			for i := range model.HTTP.Controllers {
				controller := &model.HTTP.Controllers[i]
				if controller.Type != "*"+named.Obj().Name() {
					continue
				}
				route.Path, err = httpcompiler.Path(analysis.controllerBases[controller.Type], route.Path)
				if err != nil {
					return at(pkg, d.Pos(), err)
				}
				controller.Routes = append(controller.Routes, route)
				found = true
				break
			}
			if !found {
				return at(pkg, d.Pos(), errors.New(httpReceiverMessage))
			}
		}
	}
	return nil
}

func hasHTTPPackage(model *packageModel) bool {
	if len(model.HTTP.Handlers) > 0 {
		return true
	}
	for _, controller := range model.HTTP.Controllers {
		if len(controller.Routes) > 0 {
			return true
		}
	}
	return false
}

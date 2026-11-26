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
	"go/ast"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/packages"
)

const (
	definitionMemberSeparator = "."
	errorTypeName             = "error"
	stageAnalyze              = "analyze"

	listenRequiresAReceiverMethodMessage = "listen requires a receiver method"
)

// packageAnalysis keeps typed source and its collected recipes in one invocation.
// Declaration-specific analyzers share this context without owning loading or output.
type packageAnalysis struct {
	pkg             *packages.Package
	model           *packageModel
	controllerBases map[string]string
}

func analyze(pkg *packages.Package) (*packageModel, error) {
	model := &packageModel{
		Path:    pkg.PkgPath,
		Name:    pkg.Name,
		Pkg:     pkg.Types,
		Sources: map[string]string{},
	}
	if len(pkg.GoFiles) > 0 {
		model.Dir = filepath.Dir(pkg.GoFiles[0])
	}
	analysis := packageAnalysis{
		pkg:   pkg,
		model: model,
	}
	for _, f := range pkg.Syntax {
		if err := sweepMarkers(pkg, f); err != nil {
			return nil, err
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if err := analysis.function(d); err != nil {
					return nil, err
				}
			case *ast.GenDecl:
				if err := analysis.types(d); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := analysis.attachHTTP(); err != nil {
		return nil, err
	}
	if err := attachListeners(pkg, model); err != nil {
		return nil, err
	}
	sort.Slice(model.Recipes, func(i, j int) bool {
		return model.Recipes[i].Definition.ID < model.Recipes[j].Definition.ID
	})
	return model, nil
}

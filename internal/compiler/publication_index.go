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
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
)

const (
	indexMetadataAndManifestDisagreeMessage = "index Metadata and manifest disagree"
	invalidPublicIndexPackageMessage        = "invalid public index package"
	publicIocindexIsMissingMessage          = "public iocindex is missing: "
)

func (v *publicationValidation) index() error {
	mod, data := v.module, v.data
	indexPath := filepath.Join(mod.Dir, indexSourcePath)
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return errors.New(publicIocindexIsMissingMessage + err.Error())
	}
	file, err := parser.ParseFile(token.NewFileSet(), indexPath, indexData, 0)
	if err != nil {
		return errors.New(err.Error())
	}
	if file.Name.Name != indexPackageName {
		return errors.New(invalidPublicIndexPackageMessage)
	}
	metadata := ""
	imports := v.imports
	for _, imp := range file.Imports {
		value, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return errors.New(err.Error())
		}
		imports[value] = true
	}
	for _, decl := range file.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.CONST {
			continue
		}
		for _, spec := range d.Specs {
			v, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range v.Names {
				if name.Name == indexMetadataName && i < len(v.Values) {
					lit, ok := v.Values[i].(*ast.BasicLit)
					if ok {
						metadata, err = strconv.Unquote(lit.Value)
						if err != nil {
							return errors.New(err.Error())
						}
					}
				}
			}
		}
	}
	if metadata != string(data) {
		return errors.New(indexMetadataAndManifestDisagreeMessage)
	}

	return nil
}

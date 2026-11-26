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
	"strconv"
)

const (
	indexDiagnosticFilename              = "index.gen.go"
	metadataMustBeAStringConstantMessage = "metadata must be a string constant"
	publicMetadataIsAbsentMessage        = "public Metadata is absent"
)

func indexMetadata(data []byte) ([]byte, error) {
	f, err := parser.ParseFile(token.NewFileSet(), indexDiagnosticFilename, data, 0)
	if err != nil {
		return nil, err
	}
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.CONST {
			continue
		}
		for _, spec := range d.Specs {
			v, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, n := range v.Names {
				if n.Name == indexMetadataName && i < len(v.Values) {
					lit, ok := v.Values[i].(*ast.BasicLit)
					if !ok {
						return nil, errors.New(metadataMustBeAStringConstantMessage)
					}
					value, err := strconv.Unquote(lit.Value)
					return []byte(value), err
				}
			}
		}
	}
	return nil, errors.New(publicMetadataIsAbsentMessage)
}

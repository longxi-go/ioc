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
	"strings"

	sourceannotation "github.com/longxi-go/gosr/tools/pkg/annotation"
	"github.com/longxi-go/ioc/internal/httpcompiler"
	"golang.org/x/tools/go/packages"
)

const (
	misplacedMarkerMessage = "misplaced IoC marker; injection requires an annotated component field and declarations require an adjacent doc comment"
)

// sweepMarkers rejects comments the declaration adapter would otherwise never
// visit, such as package docs and injection on unmanaged structs.
func sweepMarkers(pkg *packages.Package, file *ast.File) error {
	placed := map[*ast.CommentGroup]bool{}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			placed[d.Doc] = true
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				placed[ts.Doc] = true
				placed[d.Doc] = true
				group := sourceannotation.TypeComments(d, ts)
				a, err := readMarkers(pkg.Fset, group, declarationTarget)
				if err != nil {
					return at(pkg, ts.Pos(), err)
				}
				if st, ok := ts.Type.(*ast.StructType); ok && (a.flag(markerComponent) || a.flag(httpcompiler.Controller)) {
					for _, field := range st.Fields.List {
						placed[field.Doc] = true
					}
				}
			}
		}
	}
	for _, group := range file.Comments {
		if placed[group] {
			continue
		}
		for _, comment := range group.List {
			if strings.Contains(comment.Text, annotationPrefix) {
				if _, err := readMarkers(pkg.Fset, group, declarationTarget); err != nil {
					return at(pkg, group.Pos(), err)
				}
				return at(pkg, group.Pos(), errors.New(misplacedMarkerMessage))
			}
		}
	}
	return nil
}

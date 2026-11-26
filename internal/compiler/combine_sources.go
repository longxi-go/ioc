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
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/longxi-go/gosr/pkg/codegen/imports"
)

const (
	adapterDiagnosticFilename = "adapter.go"
	declarationSeparator      = "\n\n"
	planDiagnosticFilename    = "plan.go"
	sourceLineSeparator       = "\n"
)

func combineSources(adapter, plan []byte) ([]byte, error) {
	fs := token.NewFileSet()
	a, err := parser.ParseFile(fs, adapterDiagnosticFilename, adapter, 0)
	if err != nil {
		return nil, err
	}
	p, err := parser.ParseFile(fs, planDiagnosticFilename, plan, 0)
	if err != nil {
		return nil, err
	}
	combinedImports := imports.New("", nil)
	decls := []ast.Decl{}
	for _, file := range []*ast.File{a, p} {
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, err
			}
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			combinedImports.Add(path, alias)
		}
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
				continue
			}
			decls = append(decls, decl)
		}
	}
	e := newEmitter("")
	e.imports = combinedImports
	for _, decl := range decls {
		var b bytes.Buffer
		if err := format.Node(&b, fs, decl); err != nil {
			return nil, err
		}
		e.body += b.String() + declarationSeparator
	}
	constraint := ""
	for line := range bytes.SplitSeq(plan, []byte(sourceLineSeparator)) {
		if expression, found := bytes.CutPrefix(line, []byte(buildConstraintPrefix)); found {
			constraint = string(expression)
			break
		}
	}
	return e.finish(a.Name.Name, constraint)
}

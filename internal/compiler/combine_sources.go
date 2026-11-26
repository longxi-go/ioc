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
)

const (
	adapterDiagnosticFilename = "adapter.go"
	declarationSeparator      = "\n\n"
	planDiagnosticFilename    = "plan.go"
	sourceLineSeparator       = "\n"
)

func combineSources(adapter, plan []byte) ([]byte, error) {
	return combineNamedSources(adapter, plan, nil)
}

func combineNamedSources(adapter, plan []byte, names map[string]string) ([]byte, error) {
	fs := token.NewFileSet()
	a, err := parser.ParseFile(fs, adapterDiagnosticFilename, adapter, 0)
	if err != nil {
		return nil, err
	}
	p, err := parser.ParseFile(fs, planDiagnosticFilename, plan, 0)
	if err != nil {
		return nil, err
	}
	// Compiler-owned imports win collisions to preserve established output aliases.
	combinedImports, err := fragmentImports(fs, []*ast.File{p, a}, names)
	if err != nil {
		return nil, err
	}
	decls := []ast.Decl{}
	for _, file := range []*ast.File{a, p} {
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
				continue
			}
			decls = append(decls, decl)
		}
	}
	e := newEmitter("")

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
	view := fileView{
		Header:     generatedHeader,
		Package:    a.Name.Name,
		Constraint: constraint,
		Body:       e.body,
		Imports:    combinedImports,
	}
	source, err := executeTemplate(fileTemplate, view)
	return []byte(source), err
}

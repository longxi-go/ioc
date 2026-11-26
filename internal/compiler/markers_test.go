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
	"go/parser"
	"go/token"
	"testing"
)

func TestMarkersStrictAndFunctionAdapter(t *testing.T) {
	src := fixtureSource(t, "function-markers/fixture.go.tmpl")
	f, err := parser.ParseFile(token.NewFileSet(), "fixture.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readMarkers(nil, f.Decls[0].(*ast.FuncDecl).Doc, declarationTarget)
	if err != nil {
		t.Fatal(err)
	}
	if !got.flag("bean") || !got.flag("primary") || got.one("name") != "service" {
		t.Fatalf("wrong markers: %#v", got)
	}
	for _, marker := range []string{"+ioc:unknown", "+ioc:primary=maybe", "+ioc:bean=surprise", "+ioc:scope=", "+ioc:inject"} {
		_, err := readMarkers(nil, &ast.CommentGroup{
			List: []*ast.Comment{
				{
					Text: "// " + marker,
				},
			},
		}, declarationTarget)
		if err == nil {
			t.Errorf("accepted %s", marker)
		}
	}
}

func TestParameterMarkerQualifiersUseStructuredRegistryGrammar(t *testing.T) {
	a, err := readMarkers(nil, &ast.CommentGroup{
		List: []*ast.Comment{
			{
				Text: "// +ioc:param:parameter=read,name=readDB,qualifiers={sql,read}",
			},
		},
	}, declarationTarget)
	if err != nil {
		t.Fatal(err)
	}
	binding := a["param"][0].(parameterMarker)
	if binding.Parameter != "read" || binding.Name != "readDB" || len(binding.Qualifiers) != 2 {
		t.Fatal(binding)
	}
}

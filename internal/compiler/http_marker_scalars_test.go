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
	"go/token"
	"strconv"
	"testing"
)

func TestHTTPScalarShortcutPaths(t *testing.T) {
	for _, path := range []string{"/orders/:id", "/assets/*filepath"} {
		for _, verb := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
			for _, quoted := range []bool{false, true} {
				value := path
				if quoted {
					value = strconv.Quote(path)
				}
				text := "+ioc:http:" + verb + "=" + value
				t.Run(text, func(t *testing.T) {
					a, err := readMarkers(token.NewFileSet(), &ast.CommentGroup{
						List: []*ast.Comment{
							{
								Text: "// " + text,
							},
						},
					}, declarationTarget)
					if err != nil {
						t.Fatal(err)
					}
					route, err := httpMarker(a)
					if err != nil || route.Path != path {
						t.Fatalf("scalar route: %+v, %v", route, err)
					}
					canonical := "+ioc:http:route:method=" + route.Method + ",path=" + strconv.Quote(path)
					a, err = readMarkers(token.NewFileSet(), &ast.CommentGroup{
						List: []*ast.Comment{
							{
								Text: "// " + canonical,
							},
						},
					}, declarationTarget)
					if err != nil {
						t.Fatal(err)
					}
					want, err := httpMarker(a)
					if err != nil || route != want {
						t.Fatalf("canonical equivalence: got %+v, want %+v, %v", route, want, err)
					}
				})
			}
		}
	}
}

func TestHTTPScalarNormalizationRetainsStrictGrammar(t *testing.T) {
	for _, text := range []string{
		"+ioc:http:route:method=GET,path=\"/orders/:id\",unknown=true",
		"+ioc:http:route:method=GET,path=\"/orders/:id",
		"+ioc:http:get=\"/orders/:id\",unknown=true",
		"+ioc:http:route:method=GET,path=/orders/:id",
	} {
		t.Run(text, func(t *testing.T) {
			a, err := readMarkers(token.NewFileSet(), &ast.CommentGroup{
				List: []*ast.Comment{
					{
						Text: "// " + text,
					},
				},
			}, declarationTarget)
			if err == nil {
				_, err = httpMarker(a)
			}
			if err == nil {
				t.Fatal("malformed/unknown structured attributes accepted")
			}
		})
	}
}

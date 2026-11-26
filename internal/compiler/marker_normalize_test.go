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
	"testing"
)

func TestStructuredMarkerGrammar(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		target     markerTarget
		valid      bool
	}{
		{
			"bare",
			"+ioc:bean",
			declarationTarget,
			true,
		},
		{
			"registration",
			"+ioc:component:name=service,aliases={one,two}",
			declarationTarget,
			true,
		},
		{
			"inject",
			"+ioc:inject:name=service,qualifiers={fast}",
			fieldTarget,
			true,
		},
		{
			"property",
			"+ioc:conditional:on-property:prefix=storage,name={one,two},havingValue=\"\",matchIfMissing=true",
			declarationTarget,
			true,
		},
		{
			"profile",
			"+ioc:profile={dev,\"!prod\"}",
			declarationTarget,
			true,
		},
		{
			"legacy-name",
			"+ioc:name=service",
			declarationTarget,
			false,
		},
		{
			"legacy-alias",
			"+ioc:alias=service",
			declarationTarget,
			false,
		},
		{
			"listener-name",
			"+ioc:listener:name=service",
			declarationTarget,
			false,
		},
		{
			"placeholder",
			"+ioc:bean:name=\"${service}\"",
			declarationTarget,
			false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, err := readMarkers(token.NewFileSet(), &ast.CommentGroup{
				List: []*ast.Comment{
					{
						Text: "// " + tc.text,
					},
				},
			}, tc.target)
			if (err == nil) != tc.valid {
				t.Fatalf("annotation=%v error=%v", a, err)
			}
			if tc.name == "property" && err == nil {
				m := a[markerOnProperty][0].(propertyMarker)
				if m.HavingValue == nil || *m.HavingValue != "" || len(m.Name) != 2 {
					t.Fatalf("lost property attributes: %#v", m)
				}
			}
		})
	}
}

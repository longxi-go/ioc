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
	"slices"
	"testing"
)

func TestDependsOnMarkerSchema(t *testing.T) {
	group := &ast.CommentGroup{
		List: []*ast.Comment{
			{
				Text: "// +ioc:bean",
			},
			{
				Text: "// +ioc:depends-on=migration",
			},
			{
				Text: "// +ioc:depends-on=warmup",
			},
		},
	}
	a, err := readMarkers(nil, group, declarationTarget)
	if err != nil {
		t.Fatal(err)
	}
	names, err := compilePrerequisiteNames(a)
	if err != nil || !slices.Equal(names, []string{"migration", "warmup"}) {
		t.Fatal(names, err)
	}
	for _, marker := range []string{"auto", "before=x", "after=x", "depends-on=", "depends-on=\" \""} {
		if _, err := readMarkers(nil, &ast.CommentGroup{
			List: []*ast.Comment{
				{
					Text: "// +ioc:" + marker,
				},
			},
		}, declarationTarget); err == nil {
			t.Errorf("accepted invalid marker %s", marker)
		}
	}
	group.List[2].Text = "// +ioc:depends-on=migration"
	a, err = readMarkers(nil, group, declarationTarget)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compilePrerequisiteNames(a); err == nil {
		t.Fatal("duplicate prerequisite accepted")
	}
}

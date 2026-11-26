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
	"strconv"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

func TestDefinitionEmitsContractConstants(t *testing.T) {
	tests := []struct {
		access                                 gs.Access
		cardinality                            gs.Cardinality
		scope                                  gs.ScopeID
		accessName, cardinalityName, scopeName string
	}{
		{
			gs.Direct,
			gs.One,
			gs.Singleton,
			"Direct",
			"One",
			"Singleton",
		},
		{
			gs.Deferred,
			gs.ZeroOrOne,
			gs.Request,
			"Deferred",
			"ZeroOrOne",
			"Request",
		},
		{
			gs.Direct,
			gs.Many,
			gs.Prototype,
			"Direct",
			"Many",
			"Prototype",
		},
	}
	for _, test := range tests {
		definition := gs.BeanDefinition{
			Scope: test.scope,
			Dependencies: []gs.Dependency{
				{
					Access:      test.access,
					Cardinality: test.cardinality,
				},
			},
		}
		source, err := executeTemplate("definition", definition)
		if err != nil {
			t.Fatal(err)
		}
		expression, err := parser.ParseExpr(source)
		if err != nil {
			t.Fatal(err)
		}
		expected := map[string]string{"Access": test.accessName, "Cardinality": test.cardinalityName, "Scope": test.scopeName}
		ast.Inspect(expression, func(node ast.Node) bool {
			field, ok := node.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := field.Key.(*ast.Ident)
			if !ok || expected[key.Name] == "" {
				return true
			}
			symbol, ok := field.Value.(*ast.SelectorExpr)
			if !ok {
				t.Errorf("%s emitted %T instead of a contract constant", key.Name, field.Value)
				return true
			}
			owner, ok := symbol.X.(*ast.Ident)
			if !ok || owner.Name != "gs" || symbol.Sel.Name != expected[key.Name] {
				t.Errorf("%s emitted %#v", key.Name, symbol)
			}
			delete(expected, key.Name)
			return true
		})
		if len(expected) != 0 {
			t.Errorf("missing constant references: %v", expected)
		}
	}
}

func TestDefinitionRejectsUnknownClosedContractValues(t *testing.T) {
	tests := []gs.Dependency{
		{
			Access:      "unknown",
			Cardinality: gs.One,
		},
		{
			Access:      gs.Direct,
			Cardinality: "unknown",
		},
	}
	for _, dependency := range tests {
		_, err := executeTemplate("definition", gs.BeanDefinition{
			Scope:        gs.Singleton,
			Dependencies: []gs.Dependency{dependency},
		})
		if err == nil {
			t.Fatalf("generated unsupported dependency metadata: %#v", dependency)
		}
	}
}

func TestDefinitionPreservesCustomScope(t *testing.T) {
	const customScope = `tenant"quoted`
	source, err := executeTemplate("definition", gs.BeanDefinition{
		Scope: customScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	expression, err := parser.ParseExpr(source)
	if err != nil {
		t.Fatal(err)
	}
	definition := expression.(*ast.CompositeLit)
	for _, element := range definition.Elts {
		field := element.(*ast.KeyValueExpr)
		if field.Key.(*ast.Ident).Name != "Scope" {
			continue
		}
		conversion, ok := field.Value.(*ast.CallExpr)
		if !ok || len(conversion.Args) != 1 {
			t.Fatalf("custom scope requires an explicit ScopeID conversion: %T", field.Value)
		}
		name, ok := conversion.Fun.(*ast.SelectorExpr)
		if !ok || name.Sel.Name != "ScopeID" {
			t.Fatalf("invalid custom scope conversion: %#v", conversion.Fun)
		}
		literal, ok := conversion.Args[0].(*ast.BasicLit)
		if !ok {
			t.Fatalf("scope name is not quoted: %T", conversion.Args[0])
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil || value != customScope {
			t.Fatalf("custom scope changed: %q %v", value, err)
		}
		return
	}
	t.Fatal("scope field is absent")
}

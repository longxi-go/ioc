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
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/longxi-go/gosr/pkg/codegen/imports"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	generatedContextName    = "ctx"
	generatedValuesName     = "values"
	generatedItemValues     = "[]any{item}"
	generatedArgumentFormat = "arg%d"
)

type emitter struct {
	imports *imports.Set
	local   string
	body    string
}

func newEmitter(local string) *emitter {
	return &emitter{
		local:   local,
		imports: imports.New(local, importAlias),
	}
}

func (e *emitter) use(path, alias string) string {
	if path == e.local {
		return ""
	}
	e.imports.Add(path, alias)
	return alias + goSelector
}

func importAlias(path, _ string, _ int) string {
	return typeAlias(path)
}

func (e *emitter) typ(t types.Type) string {
	return e.imports.Type(t)
}

func (e *emitter) spec(spec typeSpec) string {
	code := spec.Code
	for path, alias := range spec.Imports {
		if path == e.local {
			code = strings.ReplaceAll(code, alias+goSelector, "")
		} else {
			e.imports.Add(path, alias)
		}
	}
	return code
}

func (e *emitter) render(name string, view any) error {
	body, err := executeTemplate(name, view)
	if err != nil {
		return err
	}
	e.body = body
	return nil
}

func (e *emitter) finish(name, constraint string) ([]byte, error) {
	view := fileView{
		Header:     generatedHeader,
		Package:    name,
		Constraint: constraint,
		Body:       e.body,
	}
	entries, err := e.imports.List()
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		view.Imports = append(view.Imports, importView{
			Path:  entry.Path,
			Alias: entry.Alias,
		})
	}
	source, err := executeTemplate(fileTemplate, view)
	return []byte(source), err
}

func (e *emitter) resolve(shape *shape, port int, ctx string) (string, error) {
	view := resolutionView{
		Provider: shape.Kind == providerShape,
		Type:     e.typ(shape.Type),
		Port:     port,
		Context:  ctx,
	}
	var err error
	if shape.Kind == providerShape {
		view.ElementType = e.typ(shape.Elem.Type)
		view.Inner, err = e.resolve(shape.Elem, port, generatedContextName)
	} else {
		view.Conversion, err = e.convert(shape, generatedValuesName, port)
	}
	if err != nil {
		return "", err
	}
	return executeTemplate(resolutionTemplate, view)
}

func (e *emitter) convert(shape *shape, values string, port int) (string, error) {
	view := conversionView{
		Optional: shape.Kind == optionalShape,
		Many:     shape.Kind == manyShape,
		Type:     e.typ(shape.Type),
		Values:   values,
	}
	var err error
	switch shape.Kind {
	case providerShape:
		return e.resolve(shape, port, generatedContextName)
	case optionalShape:
		view.Inner, err = e.convert(shape.Elem, values, port)
	case manyShape:
		view.Inner, err = e.convert(shape.Elem, generatedItemValues, port)
	}
	if err != nil {
		return "", err
	}
	return executeTemplate(conversionTemplate, view)
}

func (e *emitter) registration(recipe *recipe) (registrationView, error) {
	_, pointerResult := recipe.Result.Underlying().(*types.Pointer)
	view := registrationView{
		Definition:    recipe.Definition,
		Conditions:    recipe.Conditions,
		Result:        e.typ(recipe.Result),
		Component:     recipe.Component,
		Function:      recipe.Function,
		ReturnsError:  recipe.Error,
		PointerResult: pointerResult,
		Hooks:         recipe.Hooks,
		Listeners:     e.listenerViews(recipe),
	}
	var calls []string
	for i, argument := range recipe.Args {
		if argument.Context {
			calls = append(calls, generatedContextName)
			continue
		}
		name := fmt.Sprintf(generatedArgumentFormat, i)
		expression, err := e.resolve(argument.Shape, argument.Port, generatedContextName)
		if err != nil {
			return registrationView{}, err
		}
		calls = append(calls, name)
		view.Arguments = append(view.Arguments, argumentView{
			Name:       name,
			Expression: expression,
		})
		if argument.Field != "" {
			view.Fields = append(view.Fields, assignmentView{
				Field: argument.Field,
				Value: name,
			})
		}
	}
	view.CallArguments = strings.Join(calls, goArgumentSeparator)
	return view, nil
}

func emitPackage(model *packageModel) ([]byte, error) {
	e := newEmitter(model.Path)
	e.addRuntimeImports()
	view := packageView{
		ID:     model.Path,
		Schema: bean.SchemaVersion,
	}
	for _, recipe := range model.Recipes {
		registration, err := e.registration(recipe)
		if err != nil {
			return nil, err
		}
		view.Registrations = append(view.Registrations, registration)
	}
	if err := e.render(packageTemplate, view); err != nil {
		return nil, err
	}
	return e.finish(model.Name, model.Constraint)
}

func emitPlan(entry *packageModel, id string, recipes []publishedRecipe, indexes []string) ([]byte, error) {
	e := newEmitter(entry.Path)
	e.addRuntimeImports()
	for _, path := range indexes {
		e.use(path, imports.BlankAlias)
	}
	contracts := map[string]typeSpec{}
	queryRecipes, err := infrastructureGraph(recipes)
	if err != nil {
		return nil, err
	}
	for _, recipe := range queryRecipes {
		for i, contract := range recipe.Definition.Contracts {
			if accessibleFrom(entry.Path, recipe.Types[i]) {
				contracts[string(contract)] = recipe.Types[i]
			}
		}
	}
	keys := make([]string, 0, len(contracts))
	for contract := range contracts {
		keys = append(keys, contract)
	}
	sort.Strings(keys)
	view := queryView{
		ID: id,
	}
	for _, recipe := range recipes {
		view.DefinitionIDs = append(view.DefinitionIDs, recipe.Definition.ID)
	}
	sort.Strings(view.DefinitionIDs)
	for _, contract := range keys {
		view.Cases = append(view.Cases, queryCase{
			Type:     e.spec(contracts[contract]),
			Contract: contract,
		})
	}
	if err := e.render(planTemplate, view); err != nil {
		return nil, err
	}
	return e.finish(entry.Name, entry.Constraint)
}

func accessibleFrom(importer string, spec typeSpec) bool {
	for _, path := range spec.Hidden {
		if path != importer {
			return false
		}
	}
	for path := range spec.Imports {
		parts := strings.Split(path, importPathSeparator)
		for i, part := range parts {
			if part == internalDirectory {
				parent := strings.Join(parts[:i], importPathSeparator)
				if importer != parent && !strings.HasPrefix(importer, parent+importPathSeparator) {
					return false
				}
			}
		}
	}
	return true
}

func (e *emitter) addRuntimeImports() {
	e.use(contextPackage, contextPackage)
	e.use(contractsPackage, contractsAlias)
	e.use(beanPackage, beanAlias)
	e.use(catalogPackage, catalogAlias)
}

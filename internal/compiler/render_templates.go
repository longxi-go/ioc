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
	"embed"
	"text/template"

	"github.com/longxi-go/gosr/pkg/codegen/render"
)

const (
	fileTemplate            = "file"
	resolutionTemplate      = "resolve"
	conversionTemplate      = "convert"
	packageTemplate         = "package"
	planTemplate            = "plan"
	indexTemplate           = "index"
	stringValueTemplateFunc = "stringValue"
	templatePattern         = "templates/*.tmpl"
)

//go:embed templates/*.tmpl
var templateSources embed.FS

// Compiler-owned assets are parsed once; shared execution owns strictness and
// diagnostics. Invalid embedded assets indicate a broken compiler build.
var codeTemplates = loadCodeTemplates()

func loadCodeTemplates() *render.Set {
	set, err := render.ParseFS(templateSources, template.FuncMap{
		stringValueTemplateFunc: templateString,
		accessTemplateFunc:      templateAccess,
		cardinalityTemplateFunc: templateCardinality,
		scopeTemplateFunc:       templateScope,
	}, templatePattern)
	if err != nil {
		panic(err)
	}
	return set
}

func executeTemplate(name string, view any) (string, error) {
	data, err := codeTemplates.Execute(name, view)
	return string(data), err
}

// templateString preserves explicit empty condition values without formatting a pointer address.
func templateString(value *string) string {
	return *value
}

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

package view

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

const (
	Index           = "index.html"
	NewOrder        = "new.html"
	Detail          = "detail.html"
	Error           = "error.html"
	templatePattern = "templates/*.html"
	stylesheetPath  = "assets/style.css"
	centsPerUnit    = 100
)

//go:embed templates/*.html assets/style.css
var resources embed.FS

// Action is a permitted form operation, derived from the order state.
type Action struct {
	Path  string
	Label string
}

// Page supplies typed template data; customer input remains escaped.
type Page struct {
	Title     string
	Orders    []model.Order
	Order     model.Order
	Menu      []model.Coffee
	Actions   []Action
	CanEdit   bool
	CanDelete bool
	Error     string
	Total     int64
	Next      string
	Previous  string
}

// Renderer parses embedded templates before any HTTP request is admitted.
type Renderer struct {
	templates *template.Template
}

// New fails application startup on malformed templates.
// +ioc:bean
func New() (*Renderer, error) {
	functions := template.FuncMap{
		"money": Money,
	}
	templates, err := template.New("coffee").
		Option("missingkey=error").
		Funcs(functions).
		ParseFS(resources, templatePattern)
	if err != nil {
		return nil, err
	}
	return &Renderer{
		templates: templates,
	}, nil
}

// Money renders integer minor units without floating-point rounding.
func Money(value int64) string {
	return fmt.Sprintf("$%d.%02d", value/centsPerUnit, value%centsPerUnit)
}

// Render buffers template failures before a response header is written.
func (r *Renderer) Render(name string, page Page) ([]byte, error) {
	var output bytes.Buffer
	if err := r.templates.ExecuteTemplate(&output, name, page); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// Stylesheet returns a detached asset for the local HTTP route.
func Stylesheet() ([]byte, error) {
	return resources.ReadFile(stylesheetPath)
}

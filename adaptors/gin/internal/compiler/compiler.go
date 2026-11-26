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
	"fmt"
	"go/types"
	"strconv"
	"strings"
	"text/template"

	"github.com/longxi-go/gosr/pkg/codegen/render"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/adaptors/gin/internal/routecheck"
	protocol "github.com/longxi-go/ioc/pkg/codegen/http"
)

const (
	unsupportedScopeFormat = "%s:%d: unsupported controller scope %s"
	missingReceiverFormat  = "%s:%d: controller handler requires a receiver"
	invalidSignatureFormat = "%s:%d: Gin handler %s must have signature func(*gin.Context)"
	invalidRouteFormat     = "%s:%d: Gin route %s %s: %w"
)

const ginPath = "github.com/gin-gonic/gin"

//go:embed templates/*.tmpl
var templates embed.FS

type routeView struct {
	Method, Path, Handler, Bean, Name, Contract, Type string
}

type packageView struct {
	Name        string
	Routes      []routeView
	Entry       bool
	Packages    []string
	Controllers bool
}

// Generate returns a native typed fragment; the owning compiler alone persists it.
func Generate(p protocol.Package) ([]byte, error) {
	v := packageView{
		Name:     p.Name,
		Entry:    p.Entry,
		Packages: p.Packages,
	}
	for _, c := range p.Controllers {
		if c.Definition.Scope != gs.Singleton && c.Definition.Scope != gs.Request {
			return nil, fmt.Errorf(unsupportedScopeFormat, c.Definition.Source.File, c.Definition.Source.Line, c.Definition.Scope)
		}
		for _, r := range c.Routes {
			if err := validate(r); err != nil {
				return nil, err
			}
			if r.Signature.Recv() == nil {
				return nil, fmt.Errorf(missingReceiverFormat, r.Source.File, r.Source.Line)
			}
			contract := types.TypeString(r.Signature.Recv().Type(), func(p *types.Package) string {
				return p.Path()
			})
			if !strings.HasPrefix(contract, "*") {
				contract = "*" + contract
			}
			v.Controllers = true
			v.Routes = append(v.Routes, routeView{
				Method:   r.Method,
				Path:     r.Path,
				Handler:  r.Handler,
				Bean:     c.Definition.ID,
				Name:     c.Definition.Name,
				Contract: contract,
				Type:     c.Type,
			})
		}
	}
	for _, r := range p.Handlers {
		if err := validate(r); err != nil {
			return nil, err
		}
		v.Routes = append(v.Routes, routeView{
			Method:  r.Method,
			Path:    r.Path,
			Handler: r.Handler,
		})
	}
	set, err := render.ParseFS(templates, template.FuncMap{
		"quote": strconv.Quote,
	}, "templates/*.tmpl")
	if err != nil {
		return nil, err
	}
	return set.Execute("routes.go.tmpl", v)
}

func validate(r protocol.Route) (err error) {
	s := r.Signature
	if s == nil || s.Variadic() || s.TypeParams().Len() != 0 || s.Params().Len() != 1 || s.Results().Len() != 0 || !ginContext(s.Params().At(0).Type()) {
		return fmt.Errorf(invalidSignatureFormat, r.Source.File, r.Source.Line, r.Handler)
	}
	if err := routecheck.Validate(r.Method, r.Path); err != nil {
		return fmt.Errorf(invalidRouteFormat, r.Source.File, r.Source.Line, r.Method, r.Path, err)
	}
	return nil
}

func ginContext(t types.Type) bool {
	pointer, ok := types.Unalias(t).(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == ginPath && named.Obj().Name() == "Context"
}

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

package router

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/catalog"
	managed "github.com/longxi-go/ioc/pkg/context"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	a, err := managed.NewApplicationContext(ctx, managed.WithCatalog(catalog.New()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := a.Close(ctx); err != nil {
			t.Error(err)
		}
	}()
	bind := func(gs.BeanFactory) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.String(200, "ok")
		}
	}
	route := Route{
		Method: "GET",
		Path:   "/ok",
		Bind:   bind,
	}
	if Register(ctx, gin.New(), a, route) == nil {
		t.Fatal("accepted inactive application")
	}
	if err = a.Start(ctx); err != nil {
		t.Fatal(err)
	}
	e := gin.New()
	if err = Register(ctx, e, a, route, Route{
		Method: "GET",
		Path:   "/ok",
		Bean:   "excluded",
		Bind:   bind,
	}); err != nil {
		t.Fatal(err)
	}
	out := httptest.NewRecorder()
	e.ServeHTTP(out, httptest.NewRequest("GET", "/ok", nil))
	if out.Code != 200 || out.Body.String() != "ok" {
		t.Fatal(out)
	}
	for _, routes := range [][]Route{
		{
			route,
		},
		{
			Route{
				Method: "GET",
				Path:   "/x/:id",
				Bind:   bind,
			},
			Route{
				Method: "GET",
				Path:   "/x/:name",
				Bind:   bind,
			},
		},
		{
			Route{
				Method: "GET",
				Path:   "/x/*tail/more",
				Bind:   bind,
			},
		},
		{
			Route{
				Method: "",
				Path:   "/",
				Bind:   bind,
			},
		},
		{
			Route{
				Method: "GET",
				Path:   "/",
			},
		},
	} {
		if Register(ctx, e, a, routes...) == nil {
			t.Fatal("accepted invalid routes", routes)
		}
	}
}

func TestNativeMethods(t *testing.T) {
	ctx := context.Background()
	a, err := managed.NewApplicationContext(ctx, managed.WithCatalog(catalog.New()))
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := a.Close(ctx); err != nil {
			t.Error(err)
		}
	}()
	for _, method := range []string{"TRACE", "CONNECT", "PURGE"} {
		e := gin.New()
		r := Route{
			Method: method,
			Path:   "/",
			Bind: func(gs.BeanFactory) gin.HandlerFunc {
				return func(c *gin.Context) {
					c.Status(204)
				}
			},
		}
		if err = Register(ctx, e, a, r); err != nil {
			t.Fatal(err)
		}
		out := httptest.NewRecorder()
		e.ServeHTTP(out, httptest.NewRequest(method, "/", nil))
		if out.Code != 204 {
			t.Fatal(method, out.Code)
		}
	}
}

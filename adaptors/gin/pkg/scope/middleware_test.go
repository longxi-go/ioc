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

package scope

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gs "github.com/longxi-go/gosr/pkg/ioc"
)

type manager struct {
	fail     bool
	closed   bool
	canceled bool
	deadline bool
}

func (m *manager) OpenScope(ctx context.Context, id gs.ScopeID) (context.Context, gs.Scope, error) {
	if id != gs.Request {
		panic("wrong scope")
	}
	if m.fail {
		return nil, nil, errors.New("open")
	}
	return ctx, m, nil
}

func (m *manager) Close(ctx context.Context) error {
	m.closed = true
	m.canceled = ctx.Err() != nil
	_, m.deadline = ctx.Deadline()
	return errors.New("close")
}

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, fail := range []bool{false, true} {
		m := &manager{
			fail: fail,
		}
		var reported error
		h, err := Middleware(Options{
			Manager: m,
			OnError: func(e error) {
				reported = e
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		e := gin.New()
		e.Use(h)
		e.GET("/", func(c *gin.Context) {
			c.Status(204)
		})
		req := httptest.NewRequest("GET", "/", nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)
		out := httptest.NewRecorder()
		e.ServeHTTP(out, req)
		if fail {
			if out.Code != 503 || m.closed {
				t.Fatal(out.Code, m)
			}
		} else if out.Code != 204 || !m.closed || m.canceled || !m.deadline {
			t.Fatal(out.Code, m)
		}
		if reported == nil {
			t.Fatal("missing callback")
		}
	}
	for _, o := range []Options{
		{},
		{
			Manager: &manager{},
		},
		{
			OnError: func(error) {},
		},
		{
			Manager:        &manager{},
			OnError:        func(error) {},
			CleanupTimeout: -time.Second,
		},
	} {
		if _, err := Middleware(o); err == nil {
			t.Fatal("accepted invalid options")
		}
	}
}

func TestCleanupOnPanic(t *testing.T) {
	m := &manager{}
	var reported error
	h, err := Middleware(Options{
		Manager: m,
		OnError: func(err error) {
			reported = err
		},
		CleanupTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	e := gin.New()
	e.Use(h)
	e.GET("/", func(*gin.Context) {
		panic("handler")
	})
	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic lost")
			}
		}()
		e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}()
	if !m.closed || m.canceled || !m.deadline || reported == nil {
		t.Fatal(m, reported)
	}
}

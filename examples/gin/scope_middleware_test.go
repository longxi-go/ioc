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

package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	gs "github.com/longxi-go/gosr/pkg/ioc"
)

func TestScopeCleanupOnAbortAndPanic(t *testing.T) {
	for _, mode := range []string{"abort", "panic"} {
		t.Run(mode, func(t *testing.T) {
			app, log := startApplication(t)
			router := gin.New()
			router.Use(gin.RecoveryWithWriter(io.Discard), RequestScope(app.context, log.add))
			router.GET("/exit", func(ctx *gin.Context) {
				if _, err := app.handler.Describe(ctx.Request.Context()); err != nil {
					t.Error(err)
					ctx.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				if mode == "panic" {
					panic("test handler failure")
				}
				ctx.AbortWithStatus(http.StatusForbidden)
			})
			response := request(t.Context(), router, "/exit", "alpha")
			want := http.StatusForbidden
			if mode == "panic" {
				want = http.StatusInternalServerError
			}
			if response.Code != want || app.statistics.created.Load() != 1 || app.statistics.destroyed.Load() != 1 {
				t.Fatalf("status=%d created=%d destroyed=%d", response.Code, app.statistics.created.Load(), app.statistics.destroyed.Load())
			}
		})
	}
}

func TestCancellationAfterAllocationStillCleansScope(t *testing.T) {
	app, log := startApplication(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	router := gin.New()
	router.Use(RequestScope(app.context, log.add))
	router.GET("/cancel", func(c *gin.Context) {
		if _, err := app.handler.Describe(c.Request.Context()); err != nil {
			t.Error(err)
			return
		}
		cancel()
		_, err := app.handler.Describe(c.Request.Context())
		if !errors.Is(err, context.Canceled) {
			t.Errorf("cancellation lost: %v", err)
		}
		c.AbortWithStatus(http.StatusServiceUnavailable)
	})
	response := request(ctx, router, "/cancel", "alpha")
	if response.Code != http.StatusServiceUnavailable || app.statistics.destroyed.Load() != 1 || len(log.snapshot()) != 0 {
		t.Fatalf("status=%d destroyed=%d errors=%v", response.Code, app.statistics.destroyed.Load(), log.snapshot())
	}
}

type failingScope struct {
	err      error
	closed   bool
	canceled bool
}

func (s *failingScope) Close(ctx context.Context) error {
	s.closed = true
	s.canceled = ctx.Err() != nil
	return s.err
}

type scopeManager struct {
	scope *failingScope
}

func (s scopeManager) OpenScope(ctx context.Context, _ gs.ScopeID) (context.Context, gs.Scope, error) {
	return ctx, s.scope, nil
}

func TestCleanupFailureReportedWithoutRewritingResponse(t *testing.T) {
	failure := errors.New("cleanup failure")
	scope := &failingScope{
		err: failure,
	}
	log := &errorLog{}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	router := gin.New()
	router.Use(RequestScope(scopeManager{
		scope: scope,
	}, log.add))
	router.GET("/cleanup", func(c *gin.Context) {
		cancel()
		c.String(http.StatusAccepted, "accepted")
	})
	response := request(ctx, router, "/cleanup", "alpha")
	if response.Code != http.StatusAccepted || response.Body.String() != "accepted" || !scope.closed || scope.canceled {
		t.Fatalf("response=%d %q scope=%+v", response.Code, response.Body.String(), scope)
	}
	if !errors.Is(errors.Join(log.snapshot()...), failure) {
		t.Fatalf("cleanup failure lost: %v", log.snapshot())
	}
}

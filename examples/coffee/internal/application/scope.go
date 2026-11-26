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

package application

import (
	"context"
	"crypto/rand"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	ginscope "github.com/longxi-go/ioc/adaptors/gin/pkg/scope"
	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
)

const (
	cleanupTimeout  = 5 * time.Second
	requestIDHeader = "X-Request-ID"
	scopeFailure    = "request scope failed"
)

func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
}

// RequestScope attaches an operation ID before the adapter opens its Request scope.
func RequestScope(manager gs.ScopeManager, logger *slog.Logger) (gin.HandlerFunc, error) {
	report := func(err error) {
		logger.Error(scopeFailure, "error", err)
	}
	scoped, err := ginscope.Middleware(ginscope.Options{
		Manager:        manager,
		OnError:        report,
		CleanupTimeout: cleanupTimeout,
	})
	if err != nil {
		return nil, err
	}
	return func(ctx *gin.Context) {
		id := rand.Text()
		operation := infrastructure.WithOperation(ctx.Request.Context(), id)
		ctx.Header(requestIDHeader, id)
		ctx.Request = ctx.Request.WithContext(operation)
		scoped(ctx)
		if len(ctx.Errors) > 0 {
			logger.Error("HTTP request failed", "request_id", id, "errors", ctx.Errors.String())
		}
	}, nil
}

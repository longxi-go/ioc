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
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
)

const (
	cleanupTimeout   = 5 * time.Second
	requestIDHeader  = "X-Request-ID"
	admissionFailure = "request scope admission failed"
	cleanupFailure   = "request scope cleanup failed"
)

func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
}

// RequestScope carries only immutable metadata and closes on normal, abort and panic exits.
func RequestScope(manager gs.ScopeManager, logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := rand.Text()
		parent := infrastructure.WithOperation(ctx.Request.Context(), id)
		operation, scope, err := manager.OpenScope(parent, gs.Request)
		if err != nil {
			logger.Error(admissionFailure, "error", err)
			ctx.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		defer func() {
			cleanup, cancel := cleanupContext(operation)
			defer cancel()
			if err := scope.Close(cleanup); err != nil {
				logger.Error(cleanupFailure, "request_id", id, "error", err)
			}
		}()
		ctx.Header(requestIDHeader, id)
		ctx.Request = ctx.Request.WithContext(operation)
		ctx.Next()
		if len(ctx.Errors) > 0 {
			logger.Error("HTTP request failed", "request_id", id, "errors", ctx.Errors.String())
		}
	}
}

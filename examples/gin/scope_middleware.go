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
	"time"

	"github.com/gin-gonic/gin"
	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const cleanupTimeout = 5 * time.Second

// RequestScope closes the lifetime on every downstream exit, including panic.
// Install Gin Recovery before this middleware so unwinding releases the scope
// before Recovery writes its response. Callers must supply a valid error sink.
func RequestScope(manager gs.ScopeManager, report ErrorHandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parent := context.WithValue(ctx.Request.Context(), tenantKey{}, ctx.GetHeader(tenantHeader))
		request, scope, err := manager.OpenScope(parent, gs.Request)
		if err != nil {
			report(err)
			unavailable(ctx)
			return
		}
		defer closeScope(request, scope, report)
		ctx.Request = ctx.Request.WithContext(request)
		ctx.Next()
	}
}

func closeScope(parent context.Context, scope gs.Scope, report ErrorHandlerFunc) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(parent), cleanupTimeout)
	defer cancel()
	if err := scope.Close(cleanup); err != nil {
		report(err)
	}
}

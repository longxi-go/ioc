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
	ginscope "github.com/longxi-go/ioc/adaptors/gin/pkg/scope"
)

const cleanupTimeout = 5 * time.Second

// RequestScope attaches tenant metadata before the adapter opens its Request scope.
// Install Recovery before this middleware; the caller drains HTTP before Close.
func RequestScope(manager gs.ScopeManager, report ErrorHandlerFunc) (gin.HandlerFunc, error) {
	scoped, err := ginscope.Middleware(ginscope.Options{
		Manager: manager,
		OnError: report,
	})
	if err != nil {
		return nil, err
	}
	return func(ctx *gin.Context) {
		parent := context.WithValue(ctx.Request.Context(), tenantKey{}, ctx.GetHeader(tenantHeader))
		ctx.Request = ctx.Request.WithContext(parent)
		scoped(ctx)
		for _, failure := range ctx.Errors {
			report(failure.Err)
		}
	}, nil
}

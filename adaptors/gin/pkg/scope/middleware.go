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
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	requiredOptionsMessage = "scope Manager and OnError are required"
	negativeTimeoutMessage = "negative scope cleanup timeout"
)

const defaultCleanupTimeout = 5 * time.Second

// ErrorHandlerFunc reports scope admission and cleanup failures to the application.
// Cleanup failures may occur after a response was committed; handlers must not assume
// that replacing that response is possible.
type ErrorHandlerFunc func(error)

// Options assigns request lifetime ownership to a manager and bounded cleanup.
// Manager and OnError are required. Zero CleanupTimeout selects five seconds.
type Options struct {
	Manager        gs.ScopeManager
	OnError        ErrorHandlerFunc
	CleanupTimeout time.Duration
}

// Middleware opens a Request scope per request and closes it even during panic
// unwinding. Cleanup preserves context values but outlives request cancellation.
func Middleware(o Options) (gin.HandlerFunc, error) {
	if o.Manager == nil || o.OnError == nil {
		return nil, errors.New(requiredOptionsMessage)
	}
	if o.CleanupTimeout < 0 {
		return nil, errors.New(negativeTimeoutMessage)
	}
	if o.CleanupTimeout == 0 {
		o.CleanupTimeout = defaultCleanupTimeout
	}
	return func(c *gin.Context) {
		ctx, lifetime, err := o.Manager.OpenScope(c.Request.Context(), gs.Request)
		if err != nil {
			o.OnError(err)
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		c.Request = c.Request.WithContext(ctx)
		defer func() {
			closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), o.CleanupTimeout)
			defer cancel()
			if err := lifetime.Close(closeCtx); err != nil {
				o.OnError(err)
			}
		}()
		c.Next()
	}, nil
}

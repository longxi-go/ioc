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
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/adaptors/gin/internal/routecheck"
	managed "github.com/longxi-go/ioc/pkg/context"
)

const (
	registrationArgumentsMessage = "route registration requires context, engine and application"
	notRunningMessage            = "route registration requires a running application"
	unsupportedScopeFormat       = "controller %s has unsupported scope %s"
	invalidRouteFormat           = "invalid route %s %s"
	registrationPanicFormat      = "gin route registration: %v"
	nilHandlerFormat             = "nil handler for %s %s"
)

// HandlerBinder binds an application factory without acquiring a controller.
// Each returned handler must resolve controllers using the request context.
type HandlerBinder func(gs.BeanFactory) gin.HandlerFunc

// Route is a static declaration. Bean is the definition ID, or empty for a
// standalone function. Excluded candidates never reach Gin's route tree.
type Route struct {
	Method, Path, Bean string
	Bind               HandlerBinder
}

// Register assembles routes against the application's active definitions.
// The caller owns application startup, scope middleware, HTTP draining and close.
// Registration must finish before serving. On any error discard the engine:
// Gin cannot roll back registrations or arbitrary binder side effects.
func Register(ctx context.Context, engine *gin.Engine, application *managed.Application, routes ...Route) (err error) {
	if ctx == nil || engine == nil || application == nil {
		return errors.New(registrationArgumentsMessage)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if application.State() != gs.Running {
		return errors.New(notRunningMessage)
	}
	active := map[string]gs.BeanDefinition{}
	for _, d := range application.Definitions() {
		active[d.ID] = d
	}
	selected := make([]Route, 0, len(routes))
	for _, r := range routes {
		if r.Bean != "" {
			d, ok := active[r.Bean]
			if !ok {
				continue
			}
			if d.Scope != gs.Singleton && d.Scope != gs.Request {
				return fmt.Errorf(unsupportedScopeFormat, r.Bean, d.Scope)
			}
		}
		if r.Bind == nil || routecheck.Validate(r.Method, r.Path) != nil {
			return fmt.Errorf(invalidRouteFormat, r.Method, r.Path)
		}
		selected = append(selected, r)
	}
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf(registrationPanicFormat, p)
		}
	}()
	// A native scratch tree detects wildcard conflicts, including existing routes,
	// before any newly bound handlers are installed in the caller's engine.
	probe := gin.New()
	for _, r := range engine.Routes() {
		probe.Handle(r.Method, r.Path, probeHandler)
	}
	for _, r := range selected {
		probe.Handle(r.Method, r.Path, probeHandler)
	}
	for _, r := range selected {
		if err := ctx.Err(); err != nil {
			return err
		}
		h := r.Bind(application)
		if h == nil {
			return fmt.Errorf(nilHandlerFormat, r.Method, r.Path)
		}
		engine.Handle(r.Method, r.Path, h)
	}
	return nil
}

func probeHandler(*gin.Context) {}

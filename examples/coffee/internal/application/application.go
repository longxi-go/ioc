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
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	"github.com/longxi-go/ioc/examples/coffee/internal/controller"
	managed "github.com/longxi-go/ioc/pkg/context"
)

// Running owns the application graph. Callers drain HTTP before invoking Close.
type Running struct {
	Context  *managed.Application
	settings *configuration.Settings
	router   http.Handler
}

// Start exposes routes only after all eager Beans and prerequisites are ready.
func Start(ctx context.Context, options ...managed.Option) (_ *Running, result error) {
	owner, err := managed.NewApplicationContext(ctx, options...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if result != nil {
			cleanup, cancel := cleanupContext(ctx)
			defer cancel()
			result = errors.Join(result, owner.Close(cleanup))
		}
	}()
	if err = owner.Start(ctx); err != nil {
		return nil, err
	}
	api, err := owner.GetBean[*controller.API](ctx)
	if err != nil {
		return nil, err
	}
	pages, err := owner.GetBean[*controller.Pages](ctx)
	if err != nil {
		return nil, err
	}
	settings, err := owner.GetBean[*configuration.Settings](ctx)
	if err != nil {
		return nil, err
	}
	logger, err := owner.GetBean[*slog.Logger](ctx)
	if err != nil {
		return nil, err
	}
	router := gin.New()
	router.Use(gin.Recovery(), RequestScope(owner, logger))
	api.Register(router)
	pages.Register(router)
	return &Running{
		Context:  owner,
		settings: settings,
		router:   http.NewCrossOriginProtection().Handler(router),
	}, nil
}

func (a *Running) Router() http.Handler {
	return a.router
}

// Close joins graph cleanup; HTTP admission and drain remain the caller's responsibility.
func (a *Running) Close(ctx context.Context) error {
	return a.Context.Close(ctx)
}

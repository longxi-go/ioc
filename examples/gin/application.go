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
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/longxi-go/ioc"
	managed "github.com/longxi-go/ioc/pkg/context"
)

var errNilErrorHandler = errors.New("an error handler is required")

// Application owns the generated graph. HTTP listener shutdown remains with
// the caller, which must drain HTTP requests before closing this application.
type Application struct {
	context    *managed.Application
	router     *gin.Engine
	handler    *Handler
	statistics *Statistics
}

// New starts managed dependencies before registering routes and closes the graph on startup failure.
func New(ctx context.Context, report ErrorHandlerFunc) (_ *Application, err error) {
	if report == nil {
		return nil, errNilErrorHandler
	}
	owner, err := ioc.NewApplicationContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
			defer cancel()
			err = errors.Join(err, owner.Close(cleanup))
		}
	}()
	if err := owner.Start(ctx); err != nil {
		return nil, err
	}
	handler, err := owner.GetBean[*Handler](ctx)
	if err != nil {
		return nil, err
	}
	statistics, err := owner.GetBean[*Statistics](ctx)
	if err != nil {
		return nil, err
	}
	router := gin.New()
	router.Use(RequestScope(owner, report))
	router.GET(requestPath, handler.serve(report))
	return &Application{
		context:    owner,
		router:     router,
		handler:    handler,
		statistics: statistics,
	}, nil
}

// Router exposes the Gin handler through the standard HTTP boundary.
func (a *Application) Router() http.Handler {
	return a.router
}

// Close joins application-owned cleanup; repeated calls return the same result.
func (a *Application) Close(ctx context.Context) error {
	return a.context.Close(ctx)
}

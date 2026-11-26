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
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	requestPath        = "/request"
	requestUnavailable = "request unavailable"
)

// ErrorHandlerFunc receives operational and cleanup errors outside response data.
// It must be concurrency-safe, return promptly and must not panic.
type ErrorHandlerFunc = func(error)

// Handler receives its service through generated constructor injection.
type Handler struct {
	service *Service
}

// NewHandler keeps singleton dependencies explicit without capturing a request.
// +ioc:bean
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Describe resolves both request IDs using the same caller's scope.
func (h *Handler) Describe(ctx context.Context) (View, error) {
	return h.service.describe(ctx)
}

func (h *Handler) serve(report ErrorHandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		view, err := h.Describe(ctx.Request.Context())
		if err != nil {
			report(err)
			unavailable(ctx)
			return
		}
		ctx.JSON(http.StatusOK, view)
	}
}

type failureResponse struct {
	Error string `json:"error"`
}

func unavailable(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, failureResponse{
		Error: requestUnavailable,
	})
}

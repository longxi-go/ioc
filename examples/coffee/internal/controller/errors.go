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

package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
	"github.com/longxi-go/ioc/examples/coffee/internal/view"
)

const (
	invalidMessage     = "Invalid request. Check the fields and version."
	missingMessage     = "The order does not exist."
	conflictMessage    = "The order changed or this action is no longer allowed. Reload the order."
	unavailableMessage = "The request could not be completed."
	mediaMessage       = "Unsupported content type."
)

type errorResponse struct {
	Error string `json:"error"`
}

func classify(err error) (int, string) {
	switch {
	case errors.Is(err, model.ErrInvalid):
		return http.StatusBadRequest, invalidMessage
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound, missingMessage
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict, conflictMessage
	case errors.Is(err, errMediaType):
		return http.StatusUnsupportedMediaType, mediaMessage
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable, unavailableMessage
	default:
		return http.StatusInternalServerError, unavailableMessage
	}
}

func apiError(ctx *gin.Context, err error) {
	_ = ctx.Error(err)
	status, message := classify(err)
	ctx.JSON(status, errorResponse{
		Error: message,
	})
}

func (p *Pages) fail(ctx *gin.Context, err error) {
	_ = ctx.Error(err)
	status, message := classify(err)
	p.render(ctx, status, view.Error, view.Page{
		Title: http.StatusText(status),
		Error: message,
	})
}

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
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
	"github.com/longxi-go/ioc/examples/coffee/internal/service"
)

// API adapts JSON requests to the same use cases used by HTML forms.
// +ioc:controller
// +ioc:http:route=/api
type API struct {
	// +ioc:inject:name=orders
	orders *service.Service
}

// +ioc:http:get=/menu
func (*API) menu(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, model.Menu())
}

// +ioc:http:post=/orders
func (a *API) create(ctx *gin.Context) {
	var input detailsInput
	if err := decodeJSON(ctx, &input); err != nil {
		apiError(ctx, err)
		return
	}
	if input.Version != 0 {
		apiError(ctx, model.ErrInvalid)
		return
	}
	order, err := a.orders.Create(ctx.Request.Context(), input.Details)
	if err != nil {
		apiError(ctx, err)
		return
	}
	ctx.Header("Location", fmt.Sprintf(apiLocationFormat, order.ID))
	ctx.JSON(http.StatusCreated, order)
}

// +ioc:http:get=/orders
func (a *API) list(ctx *gin.Context) {
	query, err := listQuery(ctx)
	if err != nil {
		apiError(ctx, err)
		return
	}
	page, err := a.orders.List(ctx.Request.Context(), query)
	if err != nil {
		apiError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, page)
}

// +ioc:http:get=/orders/:id
func (a *API) get(ctx *gin.Context) {
	id, err := positive(ctx.Param(idParameter))
	if err != nil {
		apiError(ctx, err)
		return
	}
	order, err := a.orders.Get(ctx.Request.Context(), id)
	if err != nil {
		apiError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, order)
}

// +ioc:http:put=/orders/:id
func (a *API) update(ctx *gin.Context) {
	var input detailsInput
	if err := decodeJSON(ctx, &input); err != nil {
		apiError(ctx, err)
		return
	}
	token, err := mutation(ctx, input.Version)
	if err != nil {
		apiError(ctx, err)
		return
	}
	order, err := a.orders.Update(ctx.Request.Context(), model.Edit{
		Mutation: token,
		Details:  input.Details,
	})
	if err != nil {
		apiError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, order)
}

func (a *API) transition(ctx *gin.Context, target model.Status) {
	var input versionInput
	if err := decodeJSON(ctx, &input); err != nil {
		apiError(ctx, err)
		return
	}
	token, err := mutation(ctx, input.Version)
	if err != nil {
		apiError(ctx, err)
		return
	}
	order, err := a.orders.Transition(ctx.Request.Context(), model.Change{
		Mutation: token,
		To:       target,
	})
	if err != nil {
		apiError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, order)
}

// +ioc:http:delete=/orders/:id
func (a *API) delete(ctx *gin.Context) {
	version, err := positive(ctx.Query(versionField))
	if err != nil {
		apiError(ctx, err)
		return
	}
	token, err := mutation(ctx, version)
	if err != nil {
		apiError(ctx, err)
		return
	}
	if _, err = a.orders.Delete(ctx.Request.Context(), token); err != nil {
		apiError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

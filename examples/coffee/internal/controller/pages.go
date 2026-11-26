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
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
	"github.com/longxi-go/ioc/examples/coffee/internal/service"
	"github.com/longxi-go/ioc/examples/coffee/internal/view"
)

// Pages keeps HTML routing and view models outside business logic.
// +ioc:controller
// +ioc:http:route=/
type Pages struct {
	// +ioc:inject
	orders *service.Service
	// +ioc:inject
	view *view.Renderer
}

func (p *Pages) render(ctx *gin.Context, status int, name string, page view.Page) {
	body, err := p.view.Render(name, page)
	if err != nil {
		ctx.String(http.StatusInternalServerError, unavailableMessage)
		return
	}
	ctx.Data(status, HTMLContentType, body)
}

// +ioc:http:get=/assets/style.css
func (*Pages) styles(ctx *gin.Context) {
	body, err := view.Stylesheet()
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Data(http.StatusOK, CSSContentType, body)
}

// +ioc:http:get=/orders/new
func (p *Pages) newOrder(ctx *gin.Context) {
	p.render(ctx, http.StatusOK, view.NewOrder, view.Page{
		Title: "New order",
		Menu:  model.Menu(),
	})
}

func pagination(query model.Query, offset int) string {
	values := url.Values{
		limitField: {
			strconv.Itoa(query.Limit),
		},
		offsetField: {
			strconv.Itoa(offset),
		},
	}
	if query.Status != "" {
		values.Set(statusField, string(query.Status))
	}
	return IndexPath + "?" + values.Encode()
}

// +ioc:http:get=/
func (p *Pages) list(ctx *gin.Context) {
	query, err := listQuery(ctx)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	result, err := p.orders.List(ctx.Request.Context(), query)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	page := view.Page{
		Title:  "Orders",
		Orders: result.Orders,
		Total:  result.Total,
	}
	remaining := result.Total - int64(query.Offset)
	if int64(query.Limit) < remaining && query.Offset <= math.MaxInt-query.Limit {
		page.Next = pagination(query, query.Offset+query.Limit)
	}
	if query.Offset > 0 {
		page.Previous = pagination(query, max(0, query.Offset-query.Limit))
	}
	p.render(ctx, http.StatusOK, view.Index, page)
}

// +ioc:http:get=/orders/:id
func (p *Pages) get(ctx *gin.Context) {
	id, err := positive(ctx.Param(idParameter))
	if err != nil {
		p.fail(ctx, err)
		return
	}
	order, err := p.orders.Get(ctx.Request.Context(), id)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	page := view.Page{
		Title:     fmt.Sprintf("Order #%d", id),
		Order:     order,
		Menu:      model.Menu(),
		CanEdit:   order.Status == model.Placed,
		CanDelete: order.Status == model.Cancelled,
	}
	for _, action := range workflow() {
		if order.Status.CanMoveTo(action.Target) {
			page.Actions = append(page.Actions, view.Action{
				Path:  action.Path,
				Label: action.Label,
			})
		}
	}
	p.render(ctx, http.StatusOK, view.Detail, page)
}

func redirect(ctx *gin.Context, order model.Order) {
	ctx.Redirect(http.StatusSeeOther, fmt.Sprintf(orderLocationFormat, order.ID))
}

// +ioc:http:post=/orders
func (p *Pages) create(ctx *gin.Context) {
	input, err := decodeForm(ctx)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	if input.Version != 0 {
		p.fail(ctx, model.ErrInvalid)
		return
	}
	order, err := p.orders.Create(ctx.Request.Context(), input.Details)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	redirect(ctx, order)
}

// +ioc:http:post=/orders/:id/update
func (p *Pages) update(ctx *gin.Context) {
	input, err := decodeForm(ctx)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	token, err := mutation(ctx, input.Version)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	order, err := p.orders.Update(ctx.Request.Context(), model.Edit{
		Mutation: token,
		Details:  input.Details,
	})
	if err != nil {
		p.fail(ctx, err)
		return
	}
	redirect(ctx, order)
}

func (p *Pages) transition(ctx *gin.Context, target model.Status) {
	input, err := decodeForm(ctx)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	token, err := mutation(ctx, input.Version)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	order, err := p.orders.Transition(ctx.Request.Context(), model.Change{
		Mutation: token,
		To:       target,
	})
	if err != nil {
		p.fail(ctx, err)
		return
	}
	redirect(ctx, order)
}

// +ioc:http:post=/orders/:id/delete
func (p *Pages) delete(ctx *gin.Context) {
	input, err := decodeForm(ctx)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	token, err := mutation(ctx, input.Version)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	if _, err = p.orders.Delete(ctx.Request.Context(), token); err != nil {
		p.fail(ctx, err)
		return
	}
	ctx.Redirect(http.StatusSeeOther, IndexPath)
}

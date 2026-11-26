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
	"github.com/gin-gonic/gin"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

// +ioc:http:post=/orders/:id/prepare
func (a *API) prepare(ctx *gin.Context) {
	a.transition(ctx, model.Preparing)
}

// +ioc:http:post=/orders/:id/ready
func (a *API) ready(ctx *gin.Context) {
	a.transition(ctx, model.Ready)
}

// +ioc:http:post=/orders/:id/collect
func (a *API) collect(ctx *gin.Context) {
	a.transition(ctx, model.Collected)
}

// +ioc:http:post=/orders/:id/cancel
func (a *API) cancel(ctx *gin.Context) {
	a.transition(ctx, model.Cancelled)
}

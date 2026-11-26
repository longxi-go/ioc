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
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

const (
	JSONContentType = "application/json"
	HTMLContentType = "text/html; charset=utf-8"
	CSSContentType  = "text/css; charset=utf-8"
	formContentType = "application/x-www-form-urlencoded"
	maxBodyBytes    = 16 << 10
	versionField    = "version"
	customerField   = "customer"
	coffeeField     = "coffee"
	quantityField   = "quantity"
	limitField      = "limit"
	offsetField     = "offset"
	statusField     = "status"
	idParameter     = "id"
)

var errMediaType = errors.New("unsupported content type")

type detailsInput struct {
	model.Details
	Version int64 `json:"version"`
}

type versionInput struct {
	Version int64 `json:"version"`
}

func decodeJSON(ctx *gin.Context, target any) error {
	mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	if err != nil || mediaType != JSONContentType {
		return errMediaType
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBodyBytes)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return model.ErrInvalid
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return model.ErrInvalid
	}
	return nil
}

func positive(raw string) (int64, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		return 0, model.ErrInvalid
	}
	return value, nil
}

func mutation(ctx *gin.Context, version int64) (model.Mutation, error) {
	id, err := positive(ctx.Param(idParameter))
	if err != nil {
		return model.Mutation{}, err
	}
	value := model.Mutation{
		ID:      id,
		Version: version,
	}
	return value, value.Validate()
}

func listQuery(ctx *gin.Context) (model.Query, error) {
	query := model.Query{
		Status: model.Status(ctx.Query(statusField)),
		Limit:  model.DefaultLimit,
	}
	var err error
	if raw, present := ctx.GetQuery(limitField); present {
		query.Limit, err = strconv.Atoi(raw)
		if err != nil {
			return query, model.ErrInvalid
		}
	}
	if raw, present := ctx.GetQuery(offsetField); present {
		query.Offset, err = strconv.Atoi(raw)
		if err != nil {
			return query, model.ErrInvalid
		}
	}
	return query, query.Validate()
}

func decodeForm(ctx *gin.Context) (detailsInput, error) {
	mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	if err != nil || mediaType != formContentType {
		return detailsInput{}, errMediaType
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBodyBytes)
	if err := ctx.Request.ParseForm(); err != nil {
		return detailsInput{}, model.ErrInvalid
	}
	for key, values := range ctx.Request.PostForm {
		if len(values) != 1 {
			return detailsInput{}, model.ErrInvalid
		}
		switch key {
		case customerField, coffeeField, quantityField, versionField:
		default:
			return detailsInput{}, model.ErrInvalid
		}
	}
	input := detailsInput{
		Details: model.Details{
			Customer: ctx.Request.PostForm.Get(customerField),
			Coffee:   ctx.Request.PostForm.Get(coffeeField),
		},
	}
	if raw := ctx.Request.PostForm.Get(quantityField); raw != "" {
		input.Quantity, err = positive(raw)
		if err != nil {
			return input, err
		}
	}
	if raw := ctx.Request.PostForm.Get(versionField); raw != "" {
		input.Version, err = positive(raw)
		if err != nil {
			return input, err
		}
	}
	return input, nil
}

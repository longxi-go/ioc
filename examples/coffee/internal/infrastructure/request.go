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

package infrastructure

import (
	"context"
	"errors"

	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

const missingOperationMessage = "request metadata is missing"

type operationKey struct{}

// WithOperation attaches immutable metadata without retaining the HTTP framework Context.
func WithOperation(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, operationKey{}, id)
}

// NewOperation is cached once per explicitly opened HTTP request scope.
// +ioc:bean
// +ioc:scope=request
func NewOperation(ctx context.Context) (*model.Operation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, _ := ctx.Value(operationKey{}).(string)
	if id == "" {
		return nil, errors.New(missingOperationMessage)
	}
	return &model.Operation{
		ID: id,
	}, nil
}

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
	"strings"
	"sync/atomic"
)

const tenantHeader = "X-Tenant"

var errMissingTenant = errors.New("a nonblank tenant header is required")

type tenantKey struct{}

// Statistics belongs to one application and records request resource ownership.
// +ioc:component
type Statistics struct {
	created   atomic.Int64
	destroyed atomic.Int64
}

// Request owns only immutable request metadata, never a Gin or operation Context.
type Request struct {
	ID         int64
	Tenant     string
	statistics *Statistics
}

// NewRequest validates caller metadata before allocating the scoped resource.
// +ioc:bean
// +ioc:scope=request
func NewRequest(ctx context.Context, statistics *Statistics) (*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tenant, _ := ctx.Value(tenantKey{}).(string)
	if strings.TrimSpace(tenant) == "" {
		return nil, errMissingTenant
	}
	return &Request{
		ID:         statistics.created.Add(1),
		Tenant:     tenant,
		statistics: statistics,
	}, nil
}

// Destroy demonstrates cleanup independent of HTTP request cancellation.
func (r *Request) Destroy(context.Context) error {
	r.statistics.destroyed.Add(1)
	return nil
}

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

package container

import (
	"context"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

// resolution carries ownership intent through one admitted lookup tree. A lease
// is provided only by Acquire; ordinary lookups cannot own resource prototypes.
type resolution struct {
	task  *task
	lease *owner
}

// Lookup enforces lifecycle gates before validating and resolving a selector.
func (e *Engine) Lookup(ctx context.Context, s gs.Selector) (any, error) {
	values, err := e.lookup(ctx, s, false)
	if err != nil {
		return nil, err
	}
	return values[0], nil
}

// LookupAll retains all candidates ordered by Order then ID, including primaries.
func (e *Engine) LookupAll(ctx context.Context, s gs.Selector) ([]any, error) {
	return e.lookup(ctx, s, true)
}

func (e *Engine) lookup(ctx context.Context, s gs.Selector, many bool) ([]any, error) {
	call, t, finish, err := e.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	targets, err := choose(e.bound, s, many, false)
	if err != nil {
		return nil, err
	}
	return e.resolve(call, targets, resolution{
		task: t,
	})
}

func (e *Engine) resolve(ctx context.Context, targets []*bound, request resolution) ([]any, error) {
	values := make([]any, 0, len(targets))
	for _, b := range targets {
		v, err := e.get(ctx, b, request)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

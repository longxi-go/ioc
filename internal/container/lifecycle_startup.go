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

const (
	activationFailedMessage                  = "activation failed"
	missingInitializedLifecycleObjectMessage = "missing initialized lifecycle object"
)

func (e *Engine) initializeEager(ctx context.Context) error {
	bs, order, err := bind(e.recipes, e.opts)
	if err == nil {
		err = validateEnvironmentPrimary(bs)
	}
	if err == nil {
		e.events, err = newEvents(bs)
	}
	if err == nil {
		e.mu.Lock()
		e.bound = bs
		e.order = order
		e.mu.Unlock()
		for _, b := range order {
			d := b.recipe.Definition
			if d.Scope != gs.Singleton || d.Lazy {
				continue
			}
			childCall, child, finish, beginErr := e.begin(ctx)
			if beginErr != nil {
				err = beginErr
				break
			}
			_, err = e.get(childCall, b, resolution{
				task: child,
			})
			finish()
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		return err
	}
	return e.activate(ctx, order)
}

func (e *Engine) activate(ctx context.Context, order []*bound) error {
	var err error
	for _, b := range order {
		if b.recipe.Start == nil {
			continue
		}
		e.mu.Lock()
		records := append([]*allocation(nil), e.application.records...)
		e.mu.Unlock()
		var record *allocation
		for _, a := range records {
			if a.recipe.Definition.ID == b.recipe.Definition.ID {
				record = a
				break
			}
		}
		if record == nil {
			err = recipeFailure(b.recipe, gs.CreationFailed, missingInitializedLifecycleObjectMessage, nil)
			break
		}
		record.startAttempted = true
		err = invoke(func() error {
			return b.recipe.Start(ctx, record.value)
		})
		if err != nil {
			err = recipeFailure(b.recipe, gs.CreationFailed, activationFailedMessage, err)
			break
		}
		if err = ctx.Err(); err != nil {
			break
		}
	}
	return err
}

func (e *Engine) rollbackStartup(ctx context.Context, t *task) error {
	// Failure owns all callbacks launched during activation. Drain them before
	// compensating so no allocation can arrive after the rollback snapshot.
	e.mu.Lock()
	for e.active > 1 {
		changed := e.activity
		e.mu.Unlock()
		<-changed
		e.mu.Lock()
	}
	records := append([]*allocation(nil), e.application.records...)
	e.mu.Unlock()
	cleanupErr := e.cleanupRecords(context.WithoutCancel(ctx), t, records)
	e.mu.Lock()
	for _, b := range e.bound {
		b.singleton.value = nil
	}
	e.mu.Unlock()
	return cleanupErr
}

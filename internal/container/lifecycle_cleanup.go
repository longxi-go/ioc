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
	"errors"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	destroyFailedMessage = "destroy failed"
	stopFailedMessage    = "stop failed"
)

func (e *Engine) cleanupRecords(ctx context.Context, parent *task, records []*allocation) error {
	return e.cleanupPhase(ctx, cleanupRequest{
		parent:  parent,
		records: records,
		stop:    true,
		destroy: true,
	})
}

// cleanupRequest names shutdown phases and the records they own. Stop and destroy
// remain separate passes so all activations quiesce before any resource destruction.
type cleanupRequest struct {
	parent  *task
	records []*allocation
	stop    bool
	destroy bool
}

func (e *Engine) cleanupPhase(ctx context.Context, request cleanupRequest) error {
	parent, records := request.parent, request.records
	e.mu.Lock()
	t := e.addTask(parent, true, nil)
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.removeTask(t)
		e.mu.Unlock()
	}()
	ctx = context.WithValue(ctx, taskKey{}, taskToken{
		e,
		t,
	})
	var errs []error
	for i := len(records) - 1; i >= 0; i-- {
		a := records[i]
		eligible := !a.stopped && (a.startAttempted || a.recipe.Start == nil) &&
			a.recipe.Stop != nil
		if request.stop && eligible {
			a.stopped = true
			if err := invoke(func() error {
				return a.recipe.Stop(ctx, a.value)
			}); err != nil {
				errs = append(errs, recipeFailure(a.recipe, gs.CreationFailed, stopFailedMessage, err))
			}
		}
	}
	for i := len(records) - 1; i >= 0; i-- {
		a := records[i]
		if request.destroy && !a.destroyed {
			a.destroyed = true
			if a.recipe.Destroy != nil {
				if err := invoke(func() error {
					return a.recipe.Destroy(ctx, a.value)
				}); err != nil {
					errs = append(errs, recipeFailure(a.recipe, gs.CreationFailed, destroyFailedMessage, err))
				}
			}
			a.value = nil
		}
	}
	return errors.Join(errs...)
}

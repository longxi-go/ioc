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
	ownerClosedDuringAllocationMessage = "owner closed during allocation"
	recursiveCreationMessage           = "recursive creation"
)

func (e *Engine) get(ctx context.Context, b *bound, request resolution) (any, error) {
	t := request.task
	o, err := e.admitOwner(ctx, b, request)
	if err != nil {
		return nil, err
	}
	if o != nil {
		defer e.releaseOwner(o, t)
	}

	callCtx := ctx
	var cancel context.CancelFunc
	var stop afterFuncStopFunc
	if o != nil {
		callCtx, cancel = context.WithCancel(ctx)
		stop = context.AfterFunc(o.ctx, cancel)
		defer func() {
			stop()
			cancel()
		}()
	}
	e.mu.Lock()
	for ancestor := t.parent; ancestor != nil; ancestor = ancestor.parent {
		if e.tasks[ancestor] && ancestor.bean == b {
			e.mu.Unlock()
			return nil, failure(gs.DependencyCycle, b.recipe.Definition.ID, recursiveCreationMessage, nil)
		}
	}
	var s *slot
	if b.recipe.Definition.Scope == gs.Singleton {
		s = b.singleton
	} else if b.recipe.Definition.Scope != gs.Prototype {
		s = o.slots[b]
		if s == nil {
			s = &slot{}
			o.slots[b] = s
		}
	} else {
		s = &slot{}
	}
	if s.done != nil {
		return e.waitSlot(callCtx, t, s)
	}

	s.done = make(chan struct{})
	s.task = t
	t.bean = b
	e.mu.Unlock()
	value, record, err := e.create(callCtx, t, b, o)
	e.mu.Lock()
	publicationClosed := e.state == gs.Closing || e.state == gs.Closed ||
		(o != nil && o.closing) || callCtx.Err() != nil
	if err == nil && publicationClosed {
		err = callCtx.Err()
		if err == nil {
			err = failure(gs.InvalidState, b.recipe.Definition.ID, ownerClosedDuringAllocationMessage, nil)
		}
	}
	if err == nil {
		// The close gate and publication share one linearization point. No
		// closer can revoke ownership between validation and slot exposure.
		s.value = value
		s.task = nil
		t.bean = nil
		close(s.done)
		e.mu.Unlock()
		return value, nil
	}
	e.mu.Unlock()
	if record != nil {
		err = errors.Join(err, e.cleanupRecords(context.WithoutCancel(callCtx), t, []*allocation{record}))
	}
	e.mu.Lock()
	s.err = err
	s.task = nil
	t.bean = nil
	close(s.done)
	e.mu.Unlock()
	return nil, err
}

// waitSlot consumes the engine lock held by get and returns with it released.
// Both completion and cancellation remove the same wait edge before returning.
func (e *Engine) waitSlot(ctx context.Context, t *task, s *slot) (any, error) {
	if s.task == nil {
		value, err := s.value, s.err
		e.mu.Unlock()
		return value, err
	}
	if err := e.waitEdge(t, s.task); err != nil {
		e.mu.Unlock()
		return nil, err
	}
	done, creator := s.done, s.task
	e.mu.Unlock()
	select {
	case <-done:
	case <-ctx.Done():
		e.mu.Lock()
		e.unwait(t, creator)
		e.mu.Unlock()
		return nil, ctx.Err()
	}
	e.mu.Lock()
	e.unwait(t, creator)
	value, err := s.value, s.err
	e.mu.Unlock()
	return value, err
}

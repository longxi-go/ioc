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
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	scopeRequiresRunningApplicationMessage = "scope requires running application"
	unsupportedExplicitScopeMessage        = "unsupported explicit scope"
)

// OpenScope issues an owner-bound capability. Nested scope contexts retain other
// scope kinds; a foreign or closed capability cannot be transferred to this owner.
func (e *Engine) OpenScope(ctx context.Context, kind gs.ScopeID) (context.Context, gs.Scope, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != gs.Running {
		return nil, nil, failure(gs.InvalidState, "", scopeRequiresRunningApplicationMessage, nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if kind != gs.Request && !slices.Contains(e.opts.Scopes, kind) ||
		kind == gs.Singleton ||
		kind == gs.Prototype {
		return nil, nil, failure(gs.ScopeViolation, "", unsupportedExplicitScopeMessage, nil)
	}
	o := e.newOwner(kind)
	e.owners[o] = true
	previous, _ := ctx.Value(scopeKey{}).(*scopeChain)
	return context.WithValue(ctx, scopeKey{}, &scopeChain{
		owner:    o,
		previous: previous,
	}), o, nil
}

// Close releases this lifetime independently. Concurrent callers join cleanup;
// callbacks closing other scopes participate in the operation wait graph.
func (o *owner) Close(ctx context.Context) error {
	e := o.engine
	e.mu.Lock()
	caller := e.token(ctx)
	if o.done == nil {
		o.done = make(chan struct{})
		o.closing = true
		o.cancel()
		o.cleanupTask = e.addTask(nil, true, o)
		for t, count := range o.users {
			if e.waits[o.cleanupTask] == nil {
				e.waits[o.cleanupTask] = map[*task]int{}
			}
			e.waits[o.cleanupTask][t] += count
		}
		idle := o.idle
		go o.close(idle)
	}
	done := o.done
	if err := e.waitEdge(caller, o.cleanupTask); err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.unwait(caller, o.cleanupTask)
		e.mu.Unlock()
	}()
	select {
	case <-done:
		return o.result()
	default:
	}
	select {
	case <-done:
		return o.result()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (o *owner) result() error {
	o.engine.mu.Lock()
	defer o.engine.mu.Unlock()
	return o.err
}

func (o *owner) close(idle <-chan struct{}) {
	<-idle
	e := o.engine
	e.mu.Lock()
	records := o.records
	o.records = nil
	o.slots = nil
	t := o.cleanupTask
	e.mu.Unlock()
	ctx := context.WithValue(context.Background(), taskKey{}, taskToken{
		e,
		t,
	})
	err := e.cleanupRecords(ctx, t, records)
	e.mu.Lock()
	o.err = err
	delete(e.owners, o)
	e.removeTask(t)
	close(o.done)
	e.mu.Unlock()
}

// Acquire creates an explicit caller-owned lifetime for resource prototypes.
// Other scopes retain their normal ownership; closing their lease does nothing.
func (e *Engine) Acquire(ctx context.Context, s gs.Selector) (any, bean.ReleaseFunc, error) {
	call, t, finish, err := e.begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer finish()
	targets, err := choose(e.bound, s, false, false)
	if err != nil {
		return nil, nil, err
	}
	if targets[0].recipe.Definition.Scope != gs.Prototype {
		value, err := e.get(call, targets[0], resolution{
			task: t,
		})
		return value, func(context.Context) error {
			return nil
		}, err
	}
	e.mu.Lock()
	o := e.newOwner(gs.Prototype)
	e.owners[o] = true
	e.mu.Unlock()
	value, err := e.get(call, targets[0], resolution{
		task:  t,
		lease: o,
	})
	if err != nil {
		cleanupErr := o.Close(context.WithoutCancel(ctx))
		return nil, nil, errors.Join(err, cleanupErr)
	}
	return value, o.Close, nil
}

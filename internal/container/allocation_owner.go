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
	dynamicScopeEscapeMessage               = "dynamic creation captures a shorter or unrelated lifetime"
	foreignOrClosedScopeMessage             = "foreign or closed scope"
	missingExplicitScopeMessage             = "missing explicit scope"
	ownerClosingMessage                     = "owner closing"
	resourcePrototypeRequiresAcquireMessage = "resource prototype requires Acquire"
)

// ownerFor requires e.mu: scope closure and owner admission share that gate.
func (e *Engine) ownerFor(ctx context.Context, b *bound, leaseOwner *owner) (*owner, error) {
	scope := b.recipe.Definition.Scope
	if scope == gs.Singleton {
		return e.application, nil
	}
	if scope == gs.Prototype {
		if b.recipe.Stop != nil || b.recipe.Destroy != nil {
			if leaseOwner == nil {
				return nil, failure(gs.ScopeViolation, b.recipe.Definition.ID, resourcePrototypeRequiresAcquireMessage, nil)
			}
			return leaseOwner, nil
		}
		return nil, nil
	}
	chain, _ := ctx.Value(scopeKey{}).(*scopeChain)
	for chain != nil {
		if chain.owner.kind == scope {
			if chain.owner.engine != e || chain.owner.closing {
				return nil, failure(gs.ScopeViolation, b.recipe.Definition.ID, foreignOrClosedScopeMessage, nil)
			}
			return chain.owner, nil
		}
		chain = chain.previous
	}
	return nil, failure(gs.ScopeViolation, b.recipe.Definition.ID, missingExplicitScopeMessage, nil)
}

func (e *Engine) admitOwner(ctx context.Context, b *bound, request resolution) (*owner, error) {
	t := request.task
	e.mu.Lock()
	defer e.mu.Unlock()
	for ancestor := t.parent; ancestor != nil; ancestor = ancestor.parent {
		if e.tasks[ancestor] &&
			ancestor.bean != nil &&
			lifetimeEscape(ancestor.bean.recipe.Definition.Scope, b.recipe.Definition.Scope) {
			return nil, failure(gs.ScopeViolation, b.recipe.Definition.ID, dynamicScopeEscapeMessage, nil)
		}
	}
	o, err := e.ownerFor(ctx, b, request.lease)
	if err != nil {
		return nil, err
	}
	if o != nil {
		if o.closing {
			return nil, failure(gs.ScopeViolation, b.recipe.Definition.ID, ownerClosingMessage, nil)
		}
		if o.pending == 0 {
			o.idle = make(chan struct{})
		}
		o.pending++
		o.users[t]++
	}
	return o, nil
}

func (e *Engine) releaseOwner(o *owner, t *task) {
	e.mu.Lock()
	o.pending--
	o.users[t]--
	if o.users[t] == 0 {
		delete(o.users, t)
	}
	if o.cleanupTask != nil {
		e.unwait(o.cleanupTask, t)
	}
	if o.pending == 0 {
		close(o.idle)
	}
	e.mu.Unlock()
}

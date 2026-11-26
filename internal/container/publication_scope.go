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
	invalidPublicationScopeMessage = "publication cannot borrow a foreign or closed effective scope"
)

// pinPublication borrows effective scope owners atomically. It intentionally
// covers callback use after lookup returns; get alone protects only resolution.
func (e *Engine) pinPublication(ctx context.Context, t *task) (context.Context, completionFunc, error) {
	e.mu.Lock()
	if err := ctx.Err(); err != nil {
		e.mu.Unlock()
		return nil, nil, err
	}
	seen := map[gs.ScopeID]bool{}
	var owners []*owner
	chain, _ := ctx.Value(scopeKey{}).(*scopeChain)
	for node := chain; node != nil; node = node.previous {
		o := node.owner
		if seen[o.kind] {
			continue
		}
		seen[o.kind] = true
		if o.engine != e || o.closing {
			e.mu.Unlock()
			return nil, nil, failure(gs.ScopeViolation, "", invalidPublicationScopeMessage, nil)
		}
		owners = append(owners, o)
	}
	for _, o := range owners {
		if o.pending == 0 {
			o.idle = make(chan struct{})
		}
		o.pending++
		o.users[t]++
	}
	e.mu.Unlock()
	call, cancel := context.WithCancel(ctx)
	stops := make([]afterFuncStopFunc, 0, len(owners))
	for _, o := range owners {
		stops = append(stops, context.AfterFunc(o.ctx, cancel))
	}
	release := func() {
		for _, stop := range stops {
			stop()
		}
		cancel()
		e.mu.Lock()
		defer e.mu.Unlock()
		for _, o := range owners {
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
		}
	}
	return call, release, nil
}

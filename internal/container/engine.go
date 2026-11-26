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
	"slices"
	"sync"

	"github.com/longxi-go/gosr/pkg/environment"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/internal/eventdispatch"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	decoratorRequiresContractAndCallbackMessage       = "decorator requires contract and callback"
	duplicateIdentityMessage                          = "duplicate identity"
	lookupRequiresARunningOwnerMessage                = "lookup requires a running owner"
	operationWaitCycleMessage                         = "operation wait cycle"
	theApplicationEnvironmentCannotBeDecoratedMessage = "the application Environment cannot be decorated"
	theLocalEventPublisherCannotBeDecoratedMessage    = "the local event publisher cannot be decorated"
)

// Options are snapshotted before callbacks can observe the application.
type Options struct {
	Environment environment.Environment
	Snapshot    environment.Snapshot
	Scopes      []gs.ScopeID
	Decorators  []bean.Decorator
}

// Engine owns one application graph and independent instance state.
type Engine struct {
	mu           sync.Mutex
	state        gs.State
	recipes      []bean.Registration
	opts         Options
	bound, order []*bound
	base         context.Context
	cancel       context.CancelFunc
	active       int
	activity     chan struct{}
	idle         chan struct{}
	tasks        map[*task]bool
	waits        map[*task]map[*task]int
	application  *owner
	owners       map[*owner]bool
	closeDone    chan struct{}
	closeErr     error
	events       *eventdispatch.Dispatcher
	everRunning  bool
}

type task struct {
	parent  *task
	bean    *bound
	cleanup bool
	owner   *owner
}

type (
	taskKey struct{}

	taskToken struct {
		engine *Engine
		task   *task
	}
)

type slot struct {
	done  chan struct{}
	task  *task
	value any
	err   error
}

type allocation struct {
	recipe                             bean.Registration
	value                              any
	startAttempted, stopped, destroyed bool
}

type owner struct {
	engine      *Engine
	kind        gs.ScopeID
	ctx         context.Context
	cancel      context.CancelFunc
	slots       map[*bound]*slot
	users       map[*task]int
	records     []*allocation
	closing     bool
	pending     int
	idle, done  chan struct{}
	cleanupTask *task
	err         error
}

type (
	scopeKey struct{}

	scopeChain struct {
		owner    *owner
		previous *scopeChain
	}
)

// New snapshots recipes and configuration. Graph checks run at Start, before allocation.
func New(rs []bean.Registration, opts Options) (*Engine, error) {
	copied := make([]bean.Registration, len(rs))
	seen := map[string]bool{}
	for i, r := range rs {
		if err := r.Validate(); err != nil {
			return nil, err
		}
		if seen[r.Definition.ID] {
			return nil, failure(gs.DuplicateDefinition, r.Definition.ID, duplicateIdentityMessage, nil)
		}
		seen[r.Definition.ID] = true
		copied[i] = r.Clone()
	}
	opts.Scopes = slices.Clone(opts.Scopes)
	opts.Decorators = slices.Clone(opts.Decorators)
	for _, d := range opts.Decorators {
		if d.Contract == "" || d.Wrap == nil {
			return nil, failure(gs.InvalidDefinition, "", decoratorRequiresContractAndCallbackMessage, nil)
		}
		if d.Contract == bean.EnvironmentContract {
			return nil, failure(gs.InvalidDefinition, bean.EnvironmentID, theApplicationEnvironmentCannotBeDecoratedMessage, nil)
		}
		if d.Contract == bean.EventPublisherContract {
			return nil, failure(gs.InvalidDefinition, bean.EventPublisherID, theLocalEventPublisherCannotBeDecoratedMessage, nil)
		}
	}
	base, cancel := context.WithCancel(context.Background())
	idle := make(chan struct{})
	close(idle)
	e := &Engine{
		state:    gs.Created,
		recipes:  copied,
		opts:     opts,
		base:     base,
		cancel:   cancel,
		idle:     idle,
		activity: make(chan struct{}),
		tasks:    map[*task]bool{},
		waits:    map[*task]map[*task]int{},
		owners:   map[*owner]bool{},
	}
	e.application = e.newOwner(gs.Singleton)
	e.recipes = append(e.recipes, e.publisherRecipe())
	if opts.Environment != nil {
		e.recipes = append(e.recipes, e.environmentRecipe())
	}
	return e, nil
}

func (e *Engine) newOwner(kind gs.ScopeID) *owner {
	ctx, cancel := context.WithCancel(e.base)
	idle := make(chan struct{})
	close(idle)
	return &owner{
		engine: e,
		kind:   kind,
		ctx:    ctx,
		cancel: cancel,
		slots:  map[*bound]*slot{},
		users:  map[*task]int{},
		idle:   idle,
	}
}

// State is safe to observe concurrently with lifecycle operations.
func (e *Engine) State() gs.State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

// Definitions returns detached candidate or active metadata, never executable recipes.
func (e *Engine) Definitions() []gs.BeanDefinition {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.bound != nil {
		out := make([]gs.BeanDefinition, len(e.bound))
		for i, b := range e.bound {
			out[i] = b.recipe.Definition.Clone()
		}
		return out
	}
	out := make([]gs.BeanDefinition, len(e.recipes))
	for i, r := range e.recipes {
		out[i] = r.Definition.Clone()
	}
	return out
}

func (e *Engine) token(ctx context.Context) *task {
	token, _ := ctx.Value(taskKey{}).(taskToken)
	if token.engine == e && e.tasks[token.task] {
		return token.task
	}
	return nil
}

func (e *Engine) addTask(parent *task, cleanup bool, o *owner) *task {
	t := &task{
		parent:  parent,
		cleanup: cleanup,
		owner:   o,
	}
	e.tasks[t] = true
	if parent != nil {
		if e.waits[parent] == nil {
			e.waits[parent] = map[*task]int{}
		}
		e.waits[parent][t]++
	}
	return t
}

func (e *Engine) removeTask(t *task) {
	delete(e.tasks, t)
	delete(e.waits, t)
	for _, edges := range e.waits {
		delete(edges, t)
	}
}

func (e *Engine) begin(ctx context.Context) (context.Context, *task, completionFunc, error) {
	e.mu.Lock()
	parent := e.token(ctx)
	running := e.state == gs.Running
	startupLookup := e.state == gs.Starting && parent != nil
	cleanupLookup := parent != nil && parent.cleanup
	if (!running && !startupLookup) || cleanupLookup {
		e.mu.Unlock()
		return nil, nil, nil, failure(gs.InvalidState, "", lookupRequiresARunningOwnerMessage, nil)
	}
	if err := ctx.Err(); err != nil {
		e.mu.Unlock()
		return nil, nil, nil, err
	}
	if e.active == 0 {
		e.idle = make(chan struct{})
	}
	e.active++
	t := e.addTask(parent, false, nil)
	e.mu.Unlock()
	callCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(e.base, cancel)
	callCtx = context.WithValue(callCtx, taskKey{}, taskToken{
		e,
		t,
	})
	finish := func() {
		stop()
		cancel()
		e.mu.Lock()
		e.removeTask(t)
		e.active--
		close(e.activity)
		e.activity = make(chan struct{})
		if e.active == 0 {
			close(e.idle)
		}
		e.mu.Unlock()
	}
	return callCtx, t, finish, nil
}

func (e *Engine) waitEdge(from, to *task) error {
	if from == nil || to == nil {
		return nil
	}
	if e.taskReach(to, from, map[*task]bool{}) {
		return failure(gs.DependencyCycle, "", operationWaitCycleMessage, nil)
	}
	if e.waits[from] == nil {
		e.waits[from] = map[*task]int{}
	}
	e.waits[from][to]++
	return nil
}

func (e *Engine) taskReach(from, to *task, seen map[*task]bool) bool {
	if from == to {
		return true
	}
	if seen[from] {
		return false
	}
	seen[from] = true
	for next, n := range e.waits[from] {
		if n > 0 && e.taskReach(next, to, seen) {
			return true
		}
	}
	return false
}

func (e *Engine) unwait(from, to *task) {
	if edges := e.waits[from]; edges != nil {
		edges[to]--
		if edges[to] <= 0 {
			delete(edges, to)
		}
	}
}

// Check enforces the same owner and cancellation gate used by untyped lookup.
// The facade uses it before generated dispatch can report an unsupported type.
func (e *Engine) Check(ctx context.Context) error {
	_, _, finish, err := e.begin(ctx)
	if err != nil {
		return err
	}
	finish()
	return nil
}

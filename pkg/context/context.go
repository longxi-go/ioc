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

package context

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/longxi-go/gosr/pkg/environment"
	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/internal/container"
	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/ioc/pkg/catalog"
)

const (
	acquisitionQueryMustSelectExactlyOneBeanMessage = "acquisition query must select exactly one bean"
	atMostOneNameIsAcceptedMessage                  = "at most one name is accepted"
	collectionAcquisitionIsUnsupportedMessage       = "collection acquisition is unsupported"
	nilCatalogMessage                               = "nil catalog"
	nilConstructionContextMessage                   = "nil construction context"
	noGeneratedQueryPlanInstalledMessage            = "no generated query plan installed"
	queryDidNotSelectABeanMessage                   = "query did not select a bean"
	queryPanicFormat                                = "query panic: %v"
)

type configuration struct {
	catalog     *catalog.Catalog
	plan        string
	engine      container.Options
	environment environmentConfiguration
}

// Option configures composition before metadata is snapshotted.
type Option func(*configuration)

// WithCatalog selects a definition-only registry reusable across applications.
func WithCatalog(c *catalog.Catalog) Option {
	return func(o *configuration) {
		o.catalog = c
	}
}

// WithPlan explicitly selects one application closure from a shared catalog.
func WithPlan(id string) Option {
	return func(o *configuration) {
		o.plan = id
	}
}

// WithScope enables an explicit custom lifetime with the same ownership guarantees as Request.
func WithScope(scope gs.ScopeID) Option {
	return func(o *configuration) {
		o.engine.Scopes = append(o.engine.Scopes, scope)
	}
}

// WithDecorator applies static contract decoration before Init and validation.
func WithDecorator(d bean.Decorator) Option {
	return func(o *configuration) {
		o.engine.Decorators = append(o.engine.Decorators, d)
	}
}

// Application exposes managed capabilities without leaking the engine's graph or locks.
type Application struct {
	engine      *container.Engine
	query       bean.Query
	environment environment.Environment
}

var (
	_ gs.ApplicationContext  = (*Application)(nil)
	_ gs.ScopeManager        = (*Application)(nil)
	_ event.Publisher        = (*Application)(nil)
	_ gs.EnvironmentAccessor = (*Application)(nil)
)

// NewApplicationContext prepares configuration and snapshots generated contributions
// before any ordinary Bean is constructed. Configuration I/O uses ctx and may fail.
func NewApplicationContext(ctx context.Context, options ...Option) (*Application, error) {
	if ctx == nil {
		return nil, &gs.Error{
			Code:    gs.InvalidDefinition,
			Message: nilConstructionContextMessage,
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, environmentFailure(err)
	}
	config := configuration{
		catalog: catalog.Default(),
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	if config.catalog == nil {
		return nil, &gs.Error{
			Code:    gs.InvalidDefinition,
			Message: nilCatalogMessage,
		}
	}
	plan, rs, err := config.catalog.Snapshot(config.plan)
	if err != nil {
		return nil, err
	}
	owner, snapshot, err := config.environment.prepare(ctx)
	if err != nil {
		return nil, err
	}
	config.engine.Environment = owner
	config.engine.Snapshot = snapshot
	engine, err := container.New(rs, config.engine)
	if err != nil {
		return nil, err
	}
	return &Application{
		engine:      engine,
		query:       plan.Query,
		environment: owner,
	}, nil
}

// Start binds the active graph, initializes eager singletons and runs their Start
// hooks before publishing ContextRefreshed. Lazy and scoped Beans remain deferred.
func (a *Application) Start(ctx context.Context) error {
	return a.engine.Start(ctx)
}

// Close cancels work and joins tracked cleanup; cancellation only stops this wait.
func (a *Application) Close(ctx context.Context) error {
	return a.engine.Close(ctx)
}

// State reports the current lifecycle gate.
func (a *Application) State() gs.State {
	return a.engine.State()
}

// PublishEvent completes synchronous local delivery through the same owner used
// by the injected default Publisher. It is admitted only while Running.
func (a *Application) PublishEvent(ctx context.Context, value event.Event) error {
	return a.engine.PublishEvent(ctx, value)
}

// Definitions returns detached metadata for inspection without exposing recipe callbacks.
func (a *Application) Definitions() []gs.BeanDefinition {
	return a.engine.Definitions()
}

// Lookup implements the gosr named/contract lookup capability.
func (a *Application) Lookup(ctx context.Context, s gs.Selector) (any, error) {
	return a.engine.Lookup(ctx, s)
}

// LookupAll retains every matching candidate in stable metadata order.
func (a *Application) LookupAll(ctx context.Context, s gs.Selector) ([]any, error) {
	return a.engine.LookupAll(ctx, s)
}

// OpenScope binds a caller context to a lifetime owned by this application.
func (a *Application) OpenScope(ctx context.Context, scope gs.ScopeID) (context.Context, gs.Scope, error) {
	return a.engine.OpenScope(ctx, scope)
}

type queryRequest struct {
	factory gs.BeanFactory
	output  any
	names   []string
	all     bool
}

func (a *Application) dispatch(ctx context.Context, request queryRequest) (err error) {
	if err = a.engine.Check(ctx); err != nil {
		return err
	}
	if a.query == nil {
		return &gs.Error{
			Code:    gs.UnsupportedContract,
			Message: noGeneratedQueryPlanInstalledMessage,
		}
	}
	defer func() {
		if p := recover(); p != nil {
			err = &gs.Error{
				Code:    gs.CreationFailed,
				Message: fmt.Sprintf(queryPanicFormat, p),
			}
		}
	}()
	return a.query(ctx, request.factory, request.output, request.names, request.all)
}

// GetBean resolves a type published by the generated application query plan.
func (a *Application) GetBean[T any](ctx context.Context, names ...string) (T, error) {
	var value T
	if err := a.engine.Check(ctx); err != nil {
		return value, err
	}
	if len(names) > 1 {
		return value, &gs.Error{
			Code:    gs.InvalidSelector,
			Message: atMostOneNameIsAcceptedMessage,
		}
	}
	err := a.dispatch(ctx, queryRequest{
		factory: a.engine,
		output:  &value,
		names:   names,
	})
	return value, err
}

// GetBeans returns all matching T values, distinguishing this request from a []T bean.
func (a *Application) GetBeans[T any](ctx context.Context) ([]T, error) {
	var values []T
	err := a.dispatch(ctx, queryRequest{
		factory: a.engine,
		output:  &values,
		all:     true,
	})
	return values, err
}

// Provider retains only application/type/name intent; each call uses its caller's scope.
func (a *Application) Provider[T any](names ...string) gs.Provider[T] {
	names = slices.Clone(names)
	return func(ctx context.Context) (T, error) {
		return a.GetBean[T](ctx, names...)
	}
}

type acquisition struct {
	engine *container.Engine
	close  bean.ReleaseFunc
}

func (f *acquisition) Lookup(ctx context.Context, s gs.Selector) (any, error) {
	if f.close != nil {
		return nil, &gs.Error{
			Code:    gs.InvalidSelector,
			Message: acquisitionQueryMustSelectExactlyOneBeanMessage,
		}
	}
	v, close, err := f.engine.Acquire(ctx, s)
	if err == nil {
		f.close = close
	}
	return v, err
}

func (f *acquisition) LookupAll(context.Context, gs.Selector) ([]any, error) {
	return nil, &gs.Error{
		Code:    gs.InvalidSelector,
		Message: collectionAcquisitionIsUnsupportedMessage,
	}
}

type lease[T any] struct {
	value T
	close bean.ReleaseFunc
}

func (l *lease[T]) Value() T {
	return l.value
}

func (l *lease[T]) Close(ctx context.Context) error {
	return l.close(ctx)
}

// AcquireBean gives resource prototypes an explicit, repeatable cleanup handle.
func (a *Application) AcquireBean[T any](ctx context.Context, names ...string) (gs.Lease[T], error) {
	if err := a.engine.Check(ctx); err != nil {
		return nil, err
	}
	if len(names) > 1 {
		return nil, &gs.Error{
			Code:    gs.InvalidSelector,
			Message: atMostOneNameIsAcceptedMessage,
		}
	}
	var value T
	f := &acquisition{
		engine: a.engine,
	}
	err := a.dispatch(ctx, queryRequest{
		factory: f,
		output:  &value,
		names:   names,
	})
	if err != nil {
		if f.close != nil {
			err = errors.Join(err, f.close(context.WithoutCancel(ctx)))
		}
		return nil, err
	}
	if f.close == nil {
		return nil, &gs.Error{
			Code:    gs.UnsupportedContract,
			Message: queryDidNotSelectABeanMessage,
		}
	}
	return &lease[T]{
		value: value,
		close: f.close,
	}, nil
}

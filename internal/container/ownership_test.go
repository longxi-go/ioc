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
	"fmt"
	"sync/atomic"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func TestOrdinaryPrototypeAndClosedScopeReleaseReferences(t *testing.T) {
	p := recipe("p", gs.Prototype, func(context.Context, bean.Ports) (any, error) {
		return new(int), nil
	})
	r := constant("r")
	r.Definition.Scope = gs.Request
	r.Destroy = func(context.Context, any) error {
		return nil
	}
	e := boot(t, p, r)
	values := make([]any, 100)
	for i := range values {
		v, err := e.Lookup(context.Background(), gs.Selector{
			Name: "p",
		})
		if err != nil {
			t.Fatal(err)
		}
		values[i] = v
		if i > 0 && values[i] == values[i-1] {
			t.Fatal("prototype reused")
		}
	}
	e.mu.Lock()
	if len(e.application.records) != 0 || len(e.owners) != 0 {
		t.Fatal("ordinary prototypes retained")
	}
	e.mu.Unlock()
	ctx, scope, err := e.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Lookup(ctx, gs.Selector{
		Name: "r",
	}); err != nil {
		t.Fatal(err)
	}
	if err = scope.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	o := scope.(*owner)
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(o.records) != 0 ||
		len(o.slots) != 0 ||
		len(o.users) != 0 ||
		e.owners[o] {
		t.Fatal("closed scope retained instance references")
	}
}

func TestForeignCustomAndNestedScopes(t *testing.T) {
	r := constant("r")
	r.Definition.Scope = gs.Request
	c := constant("c")
	c.Definition.Scope = "tenant"
	a, err := New([]bean.Registration{r, c}, Options{
		Scopes: []gs.ScopeID{"tenant"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = a.Close(context.Background()); err != nil {
			t.Error(err)
		}
	}()
	b := boot(t, r)
	ctx, scope, err := a.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.Lookup(ctx, gs.Selector{
		Name: "r",
	}); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
	ctx, tenant, err := a.OpenScope(ctx, "tenant")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"r", "c"} {
		if _, err = a.Lookup(ctx, gs.Selector{
			Name: name,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err = scope.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = tenant.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPanicsAndFailedInitCompensateAllocation(t *testing.T) {
	for _, phase := range []string{"create", "init", "accept", "decorate"} {
		t.Run(phase, func(t *testing.T) {
			var destroyed atomic.Int32
			r := constant("a")
			r.Destroy = func(context.Context, any) error {
				destroyed.Add(1)
				return nil
			}
			opts := Options{}
			switch phase {
			case "create":
				r.Create = func(context.Context, bean.Ports) (any, error) {
					panic("factory")
				}
			case "init":
				r.Init = func(context.Context, any) error {
					panic("initializer")
				}
			case "accept":
				r.Accept = func(any) bool {
					panic("contract")
				}
			case "decorate":
				opts.Decorators = []bean.Decorator{
					{
						Contract: "a",
						Wrap: func(context.Context, any) (any, error) {
							panic("decorator")
						},
					},
				}
			}
			e, err := New([]bean.Registration{r}, opts)
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Start(context.Background()); err == nil {
				t.Fatal("panic escaped as success")
			}
			expected := int32(1)
			if phase == "create" {
				expected = 0
			}
			if destroyed.Load() != expected {
				t.Fatal("cleanup", destroyed.Load())
			}
			if err = e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
			if destroyed.Load() != expected {
				t.Fatal("double cleanup")
			}
		})
	}
}

func TestStopperOnlyAndCleanupErrorsAreAggregated(t *testing.T) {
	stopError := errors.New("stop")
	destroyError := errors.New("destroy")
	r := constant("a")
	r.Stop = func(context.Context, any) error {
		return stopError
	}
	r.Destroy = func(context.Context, any) error {
		return destroyError
	}
	e, err := New([]bean.Registration{r}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	err = e.Close(context.Background())
	if !errors.Is(err, stopError) || !errors.Is(err, destroyError) {
		t.Fatal(err)
	}
	err = e.Close(context.Background())
	if !errors.Is(err, stopError) || !errors.Is(err, destroyError) {
		t.Fatal(err)
	}
}

func TestCleanupReentrancyFailsWithoutBlocking(t *testing.T) {
	var e *Engine
	r := constant("a")
	r.Destroy = func(ctx context.Context, _ any) error {
		if err := e.Start(ctx); !errors.Is(err, &gs.Error{
			Code: gs.InvalidState,
		}) {
			return errors.New("Start callback allowed")
		}
		if err := e.Close(ctx); !errors.Is(err, &gs.Error{
			Code: gs.InvalidState,
		}) {
			return errors.New("Close callback allowed")
		}
		if _, err := e.Lookup(ctx, gs.Selector{
			Name: "a",
		}); !errors.Is(err, &gs.Error{
			Code: gs.InvalidState,
		}) {
			return errors.New("cleanup lookup allowed")
		}
		return nil
	}
	e = boot(t, r)
	if err := e.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
}

func TestPrototypeDynamicCycleRejected(t *testing.T) {
	var e *Engine
	r := recipe("p", gs.Prototype, func(ctx context.Context, _ bean.Ports) (any, error) {
		return e.Lookup(ctx, gs.Selector{
			Name: "p",
		})
	})
	e = boot(t, r)
	if _, err := e.Lookup(deadline(t), gs.Selector{
		Name: "p",
	}); !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatal(err)
	}
}

func TestFailedStartupJoinsAsynchronousOwnedCreation(t *testing.T) {
	var e *Engine
	entered := make(chan struct{})
	exited := make(chan struct{})
	var destroyed atomic.Int32
	slow := recipe("slow", gs.Singleton, func(ctx context.Context, _ bean.Ports) (any, error) {
		close(entered)
		<-ctx.Done()
		return new(int), nil
	})
	slow.Definition.Lazy = true
	slow.Destroy = func(context.Context, any) error {
		destroyed.Add(1)
		return nil
	}
	starter := constant("starter")
	starter.Start = func(ctx context.Context, _ any) error {
		go func() {
			defer close(exited)
			_, _ = e.Lookup(ctx, gs.Selector{
				Name: "slow",
			})
		}()
		<-entered
		return errors.New("start failed")
	}
	var err error
	e, err = New([]bean.Registration{starter, slow}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(deadline(t)); err == nil {
		t.Fatal("failed startup accepted")
	}
	if destroyed.Load() != 1 {
		t.Fatal("late allocation escaped startup rollback")
	}
	<-exited
	if err = e.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidDecoratorCompensatesOriginalAllocation(t *testing.T) {
	for _, kind := range []string{"wrong", "nil"} {
		t.Run(kind, func(t *testing.T) {
			resource := new(int)
			r := recipe("r", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
				return resource, nil
			})
			r.Accept = func(v any) bool {
				_, ok := v.(*int)
				return ok
			}
			r.Destroy = func(_ context.Context, v any) error {
				*v.(*int) = 1
				return nil
			}
			e, err := New([]bean.Registration{r}, Options{
				Decorators: []bean.Decorator{
					{
						Contract: "r",
						Wrap: func(context.Context, any) (any, error) {
							if kind == "nil" {
								return nil, nil
							}
							return "wrong", nil
						},
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Start(context.Background()); !errors.Is(err, &gs.Error{
				Code: gs.ContractMismatch,
			}) {
				t.Fatalf("invalid replacement accepted: %v", err)
			}
			if *resource != 1 {
				t.Fatal("original allocation escaped compensation")
			}
			if err = e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConcurrentDefinitionsDuringStart(t *testing.T) {
	rs := make([]bean.Registration, 180)
	for i := range rs {
		rs[i] = constant(fmt.Sprintf("bean-%04d", len(rs)-i))
	}
	e, err := New(rs, Options{})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- e.Start(context.Background())
	}()
	for {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
			return
		default:
			if len(e.Definitions()) != len(rs)+1 {
				t.Fatal("inconsistent metadata view")
			}
		}
	}
}

func TestCreationErrorPreservesDefinitionProvenanceAndCause(t *testing.T) {
	cause := errors.New("allocation")
	r := recipe("r", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
		return nil, cause
	})
	r.Definition.Source = gs.Source{
		File: "factory.go",
		Line: 23,
	}
	e, err := New([]bean.Registration{r}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	err = e.Start(context.Background())
	var diagnostic *gs.Error
	if !errors.Is(err, cause) ||
		!errors.As(err, &diagnostic) ||
		diagnostic.Source.File != "factory.go" ||
		diagnostic.Source.Line != 23 {
		t.Fatalf("lost creation provenance: %#v", diagnostic)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestApplicationShutdownStopsBeforeOutstandingScopeDestruction(t *testing.T) {
	var events []string
	app := constant("app")
	app.Start = func(context.Context, any) error {
		return nil
	}
	app.Stop = func(context.Context, any) error {
		events = append(events, "stop application")
		return nil
	}
	app.Destroy = func(context.Context, any) error {
		events = append(events, "destroy application")
		return nil
	}
	request := constant("request")
	request.Definition.Scope = gs.Request
	request.Destroy = func(context.Context, any) error {
		events = append(events, "destroy request")
		return nil
	}
	e := boot(t, app, request)
	ctx, _, err := e.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Lookup(ctx, gs.Selector{
		Name: "request",
	}); err != nil {
		t.Fatal(err)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(events) != "[stop application destroy request destroy application]" {
		t.Fatal(events)
	}
}

func TestFailedStartupReleasesSingletonReferences(t *testing.T) {
	r := constant("r")
	r.Start = func(context.Context, any) error {
		return errors.New("activation")
	}
	r.Destroy = func(context.Context, any) error {
		return nil
	}
	e, err := New([]bean.Registration{r}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); err == nil {
		t.Fatal("failure accepted")
	}
	e.mu.Lock()
	retained := e.bound[0].singleton.value != nil
	e.mu.Unlock()
	if retained {
		t.Fatal("failed application retained compensated singleton")
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

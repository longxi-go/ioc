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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func deadline(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestScopedFactoryCannotJoinOwnScopeClose(t *testing.T) {
	var scope gs.Scope
	r := recipe("r", gs.Request, func(ctx context.Context, _ bean.Ports) (any, error) {
		limited, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		return nil, scope.Close(limited)
	})
	e := boot(t, r)
	ctx, s, err := e.OpenScope(deadline(t), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	scope = s
	_, err = e.Lookup(ctx, gs.Selector{
		Name: "r",
	})
	if !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatalf("scope self wait must be rejected immediately: %v", err)
	}
}

func TestSiblingLookupsSharingFactoryContext(t *testing.T) {
	var e *Engine
	var calls atomic.Int32
	b := constant("b")
	b.Definition.Lazy = true
	b.Create = func(context.Context, bean.Ports) (any, error) {
		calls.Add(1)
		return new(int), nil
	}
	a := recipe("a", gs.Singleton, func(ctx context.Context, _ bean.Ports) (any, error) {
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		for range 2 {
			wg.Go(func() {
				_, err := e.Lookup(ctx, gs.Selector{
					Name: "b",
				})
				errs <- err
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				return nil, err
			}
		}
		return 1, nil
	})
	a.Definition.Lazy = true
	e = boot(t, a, b)
	if _, err := e.Lookup(deadline(t), gs.Selector{
		Name: "a",
	}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
}

func TestCrossTaskWaitCycle(t *testing.T) {
	var e *Engine
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	makeRecipe := func(id, target string) bean.Registration {
		r := recipe(id, gs.Singleton, func(ctx context.Context, _ bean.Ports) (any, error) {
			arrived <- struct{}{}
			<-release
			return e.Lookup(ctx, gs.Selector{
				Name: target,
			})
		})
		r.Definition.Lazy = true
		return r
	}
	e = boot(t, makeRecipe("a", "b"), makeRecipe("b", "a"))
	ctx := deadline(t)
	errs := make(chan error, 2)
	for _, id := range []string{"a", "b"} {
		go func() {
			_, err := e.Lookup(ctx, gs.Selector{
				Name: id,
			})
			errs <- err
		}()
	}
	<-arrived
	<-arrived
	close(release)
	for range 2 {
		if err := <-errs; !errors.Is(err, &gs.Error{
			Code: gs.DependencyCycle,
		}) {
			t.Fatal(err)
		}
	}
}

func TestCloseCancelsCreationAndRetainsCleanup(t *testing.T) {
	entered := make(chan struct{})
	destroyed := make(chan struct{})
	releaseDestroy := make(chan struct{})
	r := recipe("a", gs.Singleton, func(ctx context.Context, _ bean.Ports) (any, error) {
		close(entered)
		<-ctx.Done()
		return new(int), nil
	})
	r.Definition.Lazy = true
	r.Destroy = func(context.Context, any) error {
		close(destroyed)
		<-releaseDestroy
		return nil
	}
	e, err := New([]bean.Registration{r}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	lookup := make(chan error, 1)
	go func() {
		_, err := e.Lookup(context.Background(), gs.Selector{
			Name: "a",
		})
		lookup <- err
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = e.Close(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	<-destroyed
	if _, err = e.Lookup(context.Background(), gs.Selector{
		Name: "a",
	}); !errors.Is(err, &gs.Error{
		Code: gs.InvalidState,
	}) {
		t.Fatal(err)
	}
	close(releaseDestroy)
	if err = <-lookup; err == nil {
		t.Fatal("late object was published")
	}
	if err = e.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
	if e.State() != gs.Closed {
		t.Fatal(e.State())
	}
}

func TestIndependentScopeCleanupAndConcurrentCloseCycle(t *testing.T) {
	var first, second gs.Scope
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	r := constant("r")
	r.Definition.Scope = gs.Request
	var index atomic.Int32
	r.Create = func(context.Context, bean.Ports) (any, error) {
		return index.Add(1), nil
	}
	r.Destroy = func(ctx context.Context, v any) error {
		ready <- struct{}{}
		<-release
		if v.(int32) == 1 {
			return second.Close(ctx)
		}
		return first.Close(ctx)
	}
	e := boot(t, r)
	ctx1, s1, err := e.OpenScope(deadline(t), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	first = s1
	ctx2, s2, err := e.OpenScope(deadline(t), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	second = s2
	for _, ctx := range []context.Context{ctx1, ctx2} {
		if _, err = e.Lookup(ctx, gs.Selector{
			Name: "r",
		}); err != nil {
			t.Fatal(err)
		}
	}
	errs := make(chan error, 2)
	go func() {
		errs <- first.Close(deadline(t))
	}()
	go func() {
		errs <- second.Close(deadline(t))
	}()
	<-ready
	<-ready
	close(release)
	for range 2 {
		if err := <-errs; !errors.Is(err, &gs.Error{
			Code: gs.DependencyCycle,
		}) {
			t.Fatal(err)
		}
	}
}

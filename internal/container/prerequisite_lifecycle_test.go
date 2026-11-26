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
	"slices"
	"sync/atomic"
	"testing"
	"time"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func TestPrerequisiteNestedScopeOwnershipAndCleanup(t *testing.T) {
	var trace []string
	var next int
	p := recipe("p", gs.Request, func(context.Context, bean.Ports) (any, error) {
		next++
		return next, nil
	})
	p.Destroy = func(_ context.Context, value any) error {
		trace = append(trace, fmt.Sprint("destroy p", value))
		return nil
	}
	d := recipe("d", gs.Request, func(context.Context, bean.Ports) (any, error) {
		return next, nil
	})
	d.Definition.DependsOn = []string{"p"}
	d.Destroy = func(_ context.Context, value any) error {
		trace = append(trace, fmt.Sprint("destroy d", value))
		return nil
	}
	e := boot(t, d, p)
	outerCtx, outer, err := e.OpenScope(deadline(t), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	innerCtx, inner, err := e.OpenScope(outerCtx, gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	for i, ctx := range []context.Context{outerCtx, innerCtx, outerCtx} {
		value, err := e.Lookup(ctx, gs.Selector{
			Name: "d",
		})
		want := 1
		if i == 1 {
			want = 2
		}
		if err != nil || value != want {
			t.Fatal(value, want, err)
		}
	}
	if err := outer.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
	if err := inner.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
	if err := inner.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(trace, []string{"destroy d1", "destroy p1", "destroy d2", "destroy p2"}) {
		t.Fatal(trace)
	}
	if _, err := e.Lookup(innerCtx, gs.Selector{
		Name: "d",
	}); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
}

func TestPrerequisitePrototypeLeaseRetainsSharedSingleton(t *testing.T) {
	var trace []string
	p := constant("p")
	p.Definition.Lazy = true
	p.Destroy = func(context.Context, any) error {
		trace = append(trace, "p")
		return nil
	}
	d := constant("d")
	d.Definition.Scope = gs.Prototype
	d.Definition.DependsOn = []string{"p"}
	d.Destroy = func(context.Context, any) error {
		trace = append(trace, "d")
		return nil
	}
	e := boot(t, d, p)
	for range 2 {
		_, release, err := e.Acquire(deadline(t), gs.Selector{
			Name: "d",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err = release(deadline(t)); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(trace, []string{"d", "d"}) {
		t.Fatal("shared prerequisite released by a prototype lease", trace)
	}
	if err := e.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(trace, []string{"d", "d", "p"}) {
		t.Fatal(trace)
	}
}

func TestPrerequisiteSharedSingletonSurvivesScopedFailure(t *testing.T) {
	for _, phase := range []string{"constructor", "init"} {
		t.Run(phase, func(t *testing.T) {
			cause := errors.New("dependent failed")
			initialized, destroyed, dependentCleanup := 0, 0, 0
			p := constant("p")
			p.Definition.Lazy = true
			p.Init = func(context.Context, any) error {
				initialized++
				return nil
			}
			p.Destroy = func(context.Context, any) error {
				destroyed++
				return nil
			}
			d := recipe("d", gs.Request, func(context.Context, bean.Ports) (any, error) {
				if phase == "constructor" {
					return new(int), cause
				}
				return new(int), nil
			})
			d.Definition.DependsOn = []string{"p"}
			d.Init = func(context.Context, any) error {
				return cause
			}
			d.Destroy = func(context.Context, any) error {
				dependentCleanup++
				return nil
			}
			e := boot(t, d, p)
			for range 2 {
				ctx, scope, err := e.OpenScope(deadline(t), gs.Request)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := e.Lookup(ctx, gs.Selector{
					Name: "d",
				}); !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if err := scope.Close(deadline(t)); err != nil {
					t.Fatal(err)
				}
			}
			if initialized != 1 || destroyed != 0 || dependentCleanup != 2 {
				t.Fatal(initialized, destroyed, dependentCleanup)
			}
			if err := e.Close(deadline(t)); err != nil || destroyed != 1 {
				t.Fatal(err, destroyed)
			}
		})
	}
}

func TestPrerequisiteStartAndReverseCleanup(t *testing.T) {
	var trace []string
	a, b := constant("a"), constant("b")
	a.Definition.DependsOn = []string{"b"}
	for _, r := range []*bean.Registration{&a, &b} {
		id := r.Definition.ID
		r.Init = func(context.Context, any) error {
			trace = append(trace, "init "+id)
			return nil
		}
		r.Start = func(context.Context, any) error {
			trace = append(trace, "start "+id)
			return nil
		}
		r.Stop = func(context.Context, any) error {
			trace = append(trace, "stop "+id)
			return nil
		}
		r.Destroy = func(context.Context, any) error {
			trace = append(trace, "destroy "+id)
			return nil
		}
	}
	e := boot(t, a, b)
	if err := e.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
	want := []string{"init b", "init a", "start b", "start a", "stop a", "stop b", "destroy a", "destroy b"}
	if !slices.Equal(trace, want) {
		t.Fatal(trace)
	}
}

func TestPrerequisiteEarlyProviderStillDetectsCycle(t *testing.T) {
	a, b := constant("a"), constant("b")
	a.Definition.DependsOn = []string{"b"}
	b.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "a",
			},
			Access:      gs.Deferred,
			Cardinality: gs.One,
		},
	}
	b.Create = func(ctx context.Context, ports bean.Ports) (any, error) {
		return bean.Dependency[*int](ctx, ports, 0)
	}
	e, err := New([]bean.Registration{a, b}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Start(deadline(t)); !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatal(err)
	}
	if err := e.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
}

func TestPrerequisiteCloseCancelsInitialization(t *testing.T) {
	entered := make(chan struct{})
	var destroyed, calls atomic.Int32
	p := constant("p")
	p.Definition.Lazy = true
	p.Init = func(ctx context.Context, _ any) error {
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	}
	p.Destroy = func(context.Context, any) error {
		destroyed.Add(1)
		return nil
	}
	d := recipe("d", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
		calls.Add(1)
		return new(int), nil
	})
	d.Definition.Lazy = true
	d.Definition.DependsOn = []string{"p"}
	e := boot(t, d, p)
	result := make(chan error, 1)
	go func() {
		_, err := e.Lookup(deadline(t), gs.Selector{
			Name: "d",
		})
		result <- err
	}()
	<-entered
	if err := e.Close(deadline(t)); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err == nil || destroyed.Load() != 1 || calls.Load() != 0 {
		t.Fatal(err, destroyed.Load(), calls.Load())
	}
}

func TestPrerequisiteCanceledWaiterDoesNotCancelSharedCreator(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	p := constant("p")
	p.Definition.Lazy = true
	p.Init = func(ctx context.Context, _ any) error {
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	a, b := constant("a"), constant("b")
	for _, d := range []*bean.Registration{&a, &b} {
		d.Definition.Lazy = true
		d.Definition.DependsOn = []string{"p"}
	}
	e := boot(t, a, b, p)
	creator := make(chan error, 1)
	go func() {
		_, err := e.Lookup(deadline(t), gs.Selector{
			Name: "a",
		})
		creator <- err
	}()
	<-entered
	waitCtx, cancel := context.WithTimeout(deadline(t), 20*time.Millisecond)
	defer cancel()
	_, err := e.Lookup(waitCtx, gs.Selector{
		Name: "b",
	})
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if err := <-creator; err != nil {
		t.Fatal("waiter canceled the shared prerequisite", err)
	}
	if _, err := e.Lookup(deadline(t), gs.Selector{
		Name: "p",
	}); err != nil {
		t.Fatal("shared prerequisite was not retained", err)
	}
}

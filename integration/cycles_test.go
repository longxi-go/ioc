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

package integration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func dependency(name string, access gs.Access) gs.Dependency {
	return gs.Dependency{
		Selector: gs.Selector{
			Name: name,
		},
		Access:      access,
		Cardinality: gs.One,
	}
}

func TestSingletonCannotCaptureRequestThroughPrototype(t *testing.T) {
	a, b, c := registration("a"), registration("b"), registration("c")
	a.Definition.Dependencies = []gs.Dependency{dependency("b", gs.Direct)}
	b.Definition.Scope = gs.Prototype
	b.Definition.Dependencies = []gs.Dependency{dependency("c", gs.Direct)}
	c.Definition.Scope = gs.Request
	app := newApplication(t, []bean.Registration{a, b, c})
	if err := app.Start(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
}

func TestProviderReentryIsCycleWithoutEarlyExposure(t *testing.T) {
	a, b := registration("a"), registration("b")
	a.Definition.Dependencies = []gs.Dependency{dependency("b", gs.Deferred)}
	b.Definition.Dependencies = []gs.Dependency{dependency("a", gs.Direct)}
	b.Definition.Lazy = true
	a.Create = func(ctx context.Context, p bean.Ports) (any, error) {
		return bean.Dependency[*item](ctx, p, 0)
	}
	b.Create = func(ctx context.Context, p bean.Ports) (any, error) {
		return bean.Dependency[*item](ctx, p, 0)
	}
	app := newApplication(t, []bean.Registration{a, b})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.Start(ctx); !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatalf("reentry returned %v", err)
	}
}

func TestConcurrentProviderWaitCycleFailsWithoutTimeout(t *testing.T) {
	a, b := registration("a"), registration("b")
	a.Definition.Lazy = true
	b.Definition.Lazy = true
	a.Definition.Dependencies = []gs.Dependency{dependency("b", gs.Deferred)}
	b.Definition.Dependencies = []gs.Dependency{dependency("a", gs.Deferred)}
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	factory := func(ctx context.Context, p bean.Ports) (any, error) {
		entered <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return bean.Dependency[*item](ctx, p, 0)
	}
	a.Create = factory
	b.Create = factory
	app := newApplication(t, []bean.Registration{a, b})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for _, name := range []string{"a", "b"} {
		go func() {
			_, err := app.GetBean[*item](ctx, name)
			results <- err
		}()
	}
	for range 2 {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("constructors did not enter")
		}
	}
	close(release)
	for range 2 {
		err := <-results
		if !errors.Is(err, &gs.Error{
			Code: gs.DependencyCycle,
		}) {
			t.Fatalf("cycle returned %v", err)
		}
	}
}

func TestSiblingCallsSharingContextAreNotCycle(t *testing.T) {
	a, b := registration("a"), registration("b")
	a.Definition.Dependencies = []gs.Dependency{dependency("b", gs.Deferred)}
	b.Definition.Lazy = true
	firstEntered := make(chan struct{})
	release := make(chan struct{})
	b.Create = func(ctx context.Context, _ bean.Ports) (any, error) {
		close(firstEntered)
		select {
		case <-release:
			return &item{
				name: "b",
			}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	a.Create = func(ctx context.Context, p bean.Ports) (any, error) {
		results := make(chan error, 2)
		var tasks sync.WaitGroup
		tasks.Go(func() {
			_, err := bean.Dependency[*item](ctx, p, 0)
			results <- err
		})
		select {
		case <-firstEntered:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		tasks.Go(func() {
			_, err := bean.Dependency[*item](ctx, p, 0)
			results <- err
		})
		close(release)
		tasks.Wait()
		first, second := <-results, <-results
		return &item{
			name: "a",
		}, errors.Join(first, second)
	}
	app := newApplication(t, []bean.Registration{a, b})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
}

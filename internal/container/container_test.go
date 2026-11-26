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
	"sync"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func recipe(id string, scope gs.ScopeID, create bean.Factory) bean.Registration {
	return bean.Registration{
		Definition: gs.BeanDefinition{
			ID:        id,
			Name:      id,
			Contracts: []gs.Contract{gs.Contract(id)},
			Scope:     scope,
		},
		Create: create,
	}
}

func constant(id string) bean.Registration {
	return recipe(id, gs.Singleton, func(context.Context, bean.Ports) (any, error) {
		return new(int), nil
	})
}

func boot(t *testing.T, rs ...bean.Registration) *Engine {
	t.Helper()
	e, err := New(rs, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return e
}

func TestLifecycleIdentityAndOrder(t *testing.T) {
	var events []string
	a := constant("a")
	b := constant("b")
	b.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "a",
			},
			Access:      gs.Direct,
			Cardinality: gs.One,
		},
	}
	for _, r := range []*bean.Registration{&a, &b} {
		id := r.Definition.ID
		r.Init = func(context.Context, any) error {
			events = append(events, "init "+id)
			return nil
		}
		r.Start = func(context.Context, any) error {
			events = append(events, "start "+id)
			return nil
		}
		r.Stop = func(context.Context, any) error {
			events = append(events, "stop "+id)
			return nil
		}
		r.Destroy = func(context.Context, any) error {
			events = append(events, "destroy "+id)
			return nil
		}
	}
	e := boot(t, b, a)
	x, err := e.Lookup(context.Background(), gs.Selector{
		Name: "a",
	})
	if err != nil {
		t.Fatal(err)
	}
	y, err := e.Lookup(context.Background(), gs.Selector{
		Name: "a",
	})
	if err != nil || x != y {
		t.Fatal("singleton identity", err)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "[init a init b start a start b stop b stop a destroy b destroy a]"
	if fmt.Sprint(events) != want {
		t.Fatal(events)
	}
}

func TestGraphValidationBeforeAllocation(t *testing.T) {
	for _, code := range []gs.Code{gs.MissingBean, gs.DependencyCycle, gs.ScopeViolation} {
		t.Run(string(code), func(t *testing.T) {
			calls := 0
			a := recipe("a", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
				calls++
				return 1, nil
			})
			a.Definition.Dependencies = []gs.Dependency{
				{
					Selector: gs.Selector{
						Name: "b",
					},
					Access:      gs.Direct,
					Cardinality: gs.One,
				},
			}
			b := constant("b")
			switch code {
			case gs.MissingBean:
				b.Definition.Name = "other"
			case gs.DependencyCycle:
				b.Definition.Dependencies = []gs.Dependency{
					{
						Selector: gs.Selector{
							Name: "a",
						},
						Access:      gs.Direct,
						Cardinality: gs.One,
					},
				}
			case gs.ScopeViolation:
				b.Definition.Scope = gs.Request
			}
			e, err := New([]bean.Registration{a, b}, Options{})
			if err != nil {
				t.Fatal(err)
			}
			err = e.Start(context.Background())
			if !errors.Is(err, &gs.Error{
				Code: code,
			}) || calls != 0 {
				t.Fatal(calls, err)
			}
			if err = e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStickyFailureAndConcurrentSingleton(t *testing.T) {
	calls := 0
	var mu sync.Mutex
	r := constant("a")
	r.Definition.Lazy = true
	r.Create = func(context.Context, bean.Ports) (any, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return nil, errors.New("creation")
	}
	e := boot(t, r)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			_, err := e.Lookup(context.Background(), gs.Selector{
				Name: "a",
			})
			if !errors.Is(err, &gs.Error{
				Code: gs.CreationFailed,
			}) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if calls != 1 {
		t.Fatal(calls)
	}
}

func TestScopeIsolationAndPrototypeLease(t *testing.T) {
	r := constant("request")
	r.Definition.Scope = gs.Request
	p := constant("prototype")
	p.Definition.Scope = gs.Prototype
	destroyed := 0
	p.Destroy = func(context.Context, any) error {
		destroyed++
		return nil
	}
	e := boot(t, r, p)
	ctx, s, err := e.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	a, err := e.Lookup(ctx, gs.Selector{
		Name: "request",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Lookup(ctx, gs.Selector{
		Name: "request",
	})
	if err != nil || a != b {
		t.Fatal(err)
	}
	if err = s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Lookup(ctx, gs.Selector{
		Name: "request",
	}); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
	if _, err = e.Lookup(context.Background(), gs.Selector{
		Name: "prototype",
	}); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
	v, closeLease, err := e.Acquire(context.Background(), gs.Selector{
		Name: "prototype",
	})
	if err != nil || v == nil {
		t.Fatal(err)
	}
	if err = closeLease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = closeLease(context.Background()); err != nil || destroyed != 1 {
		t.Fatal(destroyed, err)
	}
}

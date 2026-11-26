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
	"sync/atomic"
	"testing"
	"time"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

type cycleParent struct {
	provider gs.Provider[*cycleChild]
}

type cycleChild struct {
	a *cycleParent
}

func TestProviderCycleTiming(t *testing.T) {
	for _, phase := range []string{"stored", "constructor", "init", "start"} {
		t.Run(phase, func(t *testing.T) {
			a := recipe("a", gs.Singleton, func(ctx context.Context, p bean.Ports) (any, error) {
				value := &cycleParent{
					provider: func(call context.Context) (*cycleChild, error) {
						return bean.Dependency[*cycleChild](call, p, 0)
					},
				}
				if phase == "constructor" {
					if _, err := value.provider(ctx); err != nil {
						return value, err
					}
				}
				return value, nil
			})
			a.Definition.Dependencies = []gs.Dependency{
				{
					Selector: gs.Selector{
						Name: "b",
					},
					Access:      gs.Deferred,
					Cardinality: gs.One,
				},
			}
			if phase == "init" {
				a.Init = func(ctx context.Context, value any) error {
					_, err := value.(*cycleParent).provider(ctx)
					return err
				}
			}
			if phase == "start" {
				a.Start = func(ctx context.Context, value any) error {
					_, err := value.(*cycleParent).provider(ctx)
					return err
				}
			}
			b := recipe("b", gs.Singleton, func(ctx context.Context, p bean.Ports) (any, error) {
				parent, err := bean.Dependency[*cycleParent](ctx, p, 0)
				return &cycleChild{
					a: parent,
				}, err
			})
			b.Definition.Dependencies = []gs.Dependency{
				{
					Selector: gs.Selector{
						Name: "a",
					},
					Access:      gs.Direct,
					Cardinality: gs.One,
				},
			}
			e, err := New([]bean.Registration{a, b}, Options{})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = e.Start(ctx)
			if phase == "constructor" || phase == "init" {
				if !errors.Is(err, &gs.Error{
					Code: gs.DependencyCycle,
				}) {
					t.Fatalf("expected cycle, got %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				raw, err := e.Lookup(ctx, gs.Selector{
					Name: "a",
				})
				if err != nil {
					t.Fatal(err)
				}
				value, err := raw.(*cycleParent).provider(ctx)
				if err != nil || value.a != raw {
					t.Fatalf("round trip: %v %v", value, err)
				}
			}
			if err := e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLazyDoesNotBreakDirectCycle(t *testing.T) {
	a, b := constant("a"), constant("b")
	a.Definition.Lazy = true
	b.Definition.Lazy = true
	a.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "b",
			},
			Access:      gs.Direct,
			Cardinality: gs.One,
		},
	}
	b.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "a",
			},
			Access:      gs.Direct,
			Cardinality: gs.One,
		},
	}
	e, err := New([]bean.Registration{a, b}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatal(err)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDeferredRequiredMissingStillFailsStart(t *testing.T) {
	a := constant("a")
	a.Definition.Lazy = true
	a.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "absent",
			},
			Access:      gs.Deferred,
			Cardinality: gs.One,
		},
	}
	e, err := New([]bean.Registration{a}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.MissingBean,
	}) {
		t.Fatal(err)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDeferredDoesNotGuaranteeActivationOrder(t *testing.T) {
	var targetStarted atomic.Bool
	b := constant("b")
	b.Start = func(context.Context, any) error {
		targetStarted.Store(true)
		return nil
	}
	a := recipe("a", gs.Singleton, func(_ context.Context, p bean.Ports) (any, error) {
		return gs.Provider[*int](func(ctx context.Context) (*int, error) {
			return bean.Dependency[*int](ctx, p, 0)
		}), nil
	})
	a.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "b",
			},
			Access:      gs.Deferred,
			Cardinality: gs.One,
		},
	}
	a.Start = func(ctx context.Context, value any) error {
		if _, err := value.(gs.Provider[*int])(ctx); err != nil {
			return err
		}
		if targetStarted.Load() {
			t.Fatal("expected initialized but not yet activated target")
		}
		return nil
	}
	e := boot(t, a, b)
	if !targetStarted.Load() || e.State() != gs.Running {
		t.Fatal("activation did not finish")
	}
}

func TestCallbackContextCancellation(t *testing.T) {
	contexts := map[string]context.Context{}
	r := recipe("a", gs.Singleton, func(ctx context.Context, _ bean.Ports) (any, error) {
		contexts["constructor"] = ctx
		return new(int), nil
	})
	r.Init = func(ctx context.Context, _ any) error {
		contexts["init"] = ctx
		return nil
	}
	r.Start = func(ctx context.Context, _ any) error {
		contexts["start"] = ctx
		return nil
	}
	r.Stop = func(ctx context.Context, _ any) error {
		contexts["stop"] = ctx
		return nil
	}
	r.Destroy = func(ctx context.Context, _ any) error {
		contexts["destroy"] = ctx
		return nil
	}
	e := boot(t, r)
	for _, phase := range []string{"constructor", "init", "start"} {
		if contexts[phase].Err() != context.Canceled {
			t.Fatalf("%s: %v", phase, contexts[phase].Err())
		}
	}
	if err := e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"stop", "destroy"} {
		if contexts[phase].Err() != nil || contexts[phase].Done() != nil {
			t.Fatalf("%s expected noncancelable cleanup context", phase)
		}
	}
}

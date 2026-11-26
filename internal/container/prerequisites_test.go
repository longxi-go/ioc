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
	"sync"
	"sync/atomic"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func TestPrerequisiteLazyInitializationBeforeFactory(t *testing.T) {
	for _, lazyDependent := range []bool{false, true} {
		t.Run(fmt.Sprint(lazyDependent), func(t *testing.T) {
			var trace []string
			p := constant("migration")
			p.Definition.Lazy = true
			p.Definition.Aliases = []string{"schema"}
			p.Init = func(context.Context, any) error {
				trace = append(trace, "migration init")
				return nil
			}
			d := recipe("repository", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
				trace = append(trace, "repository factory")
				return new(int), nil
			})
			d.Definition.DependsOn = []string{"schema"}
			d.Definition.Lazy = lazyDependent
			e := boot(t, d, p)
			if lazyDependent && len(trace) != 0 {
				t.Fatal("lazy dependent activated prerequisites", trace)
			}
			if _, err := e.Lookup(context.Background(), gs.Selector{
				Name: "repository",
			}); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(trace, []string{"migration init", "repository factory"}) {
				t.Fatal(trace)
			}
		})
	}
}

func TestPrerequisiteBindingFailuresBeforeAllocation(t *testing.T) {
	for _, kind := range []string{"missing", "excluded", "duplicate alias", "mixed cycle"} {
		t.Run(kind, func(t *testing.T) {
			var calls int
			d := recipe("d", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
				calls++
				return new(int), nil
			})
			d.Definition.DependsOn = []string{"p"}
			p := constant("p")
			want := gs.MissingBean
			switch kind {
			case "missing":
				p.Definition.Name = "other"
			case "excluded":
				p.Conditions = []bean.Condition{
					{
						Property: &bean.PropertyCondition{
							Names:       []string{"enabled"},
							HavingValue: new("true"),
						},
					},
				}
			case "duplicate alias":
				p.Definition.Aliases = []string{"alias"}
				d.Definition.DependsOn = []string{"p", "alias"}
				want = gs.InvalidDefinition
			case "mixed cycle":
				p.Definition.Dependencies = []gs.Dependency{
					{
						Selector: gs.Selector{
							Name: "d",
						},
						Access:      gs.Direct,
						Cardinality: gs.One,
					},
				}
				want = gs.DependencyCycle
			}
			e, err := New([]bean.Registration{d, p}, Options{
				Snapshot: conditionSnapshot(t, nil),
			})
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Start(context.Background()); !errors.Is(err, &gs.Error{
				Code: want,
			}) || calls != 0 {
				t.Fatalf("calls=%d, want %s, got %v", calls, want, err)
			}
			if err := e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPrerequisiteScopeMatrix(t *testing.T) {
	for _, from := range []gs.ScopeID{gs.Singleton, gs.Request, "tenant", gs.Prototype} {
		for _, to := range []gs.ScopeID{gs.Singleton, gs.Request, "tenant", gs.Prototype} {
			t.Run(string(from)+"/"+string(to), func(t *testing.T) {
				d, p := constant("d"), constant("p")
				d.Definition.Scope, p.Definition.Scope = from, to
				d.Definition.DependsOn = []string{"p"}
				e, err := New([]bean.Registration{d, p}, Options{
					Scopes: []gs.ScopeID{"tenant"},
				})
				if err != nil {
					t.Fatal(err)
				}
				err = e.Start(context.Background())
				allowed := to == gs.Singleton || from == to && from != gs.Prototype
				if allowed && err != nil || !allowed && !errors.Is(err, &gs.Error{
					Code: gs.ScopeViolation,
				}) {
					t.Fatalf("allowed=%v, got %v", allowed, err)
				}
				if err := e.Close(context.Background()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestPrerequisiteInitFailurePreventsFactory(t *testing.T) {
	cause := errors.New("migration failed")
	var calls int
	p := constant("p")
	p.Definition.Lazy = true
	p.Init = func(context.Context, any) error {
		return cause
	}
	d := recipe("d", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
		calls++
		return new(int), nil
	})
	d.Definition.DependsOn = []string{"p"}
	e, err := New([]bean.Registration{d, p}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); !errors.Is(err, cause) || calls != 0 {
		t.Fatal(calls, err)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPrerequisiteConcurrentSharedInitialization(t *testing.T) {
	var calls atomic.Int32
	p := constant("p")
	p.Definition.Lazy = true
	p.Init = func(context.Context, any) error {
		calls.Add(1)
		return nil
	}
	a, b := constant("a"), constant("b")
	for _, d := range []*bean.Registration{&a, &b} {
		d.Definition.Lazy = true
		d.Definition.DependsOn = []string{"p"}
	}
	e := boot(t, a, b, p)
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Go(func() {
			name := "a"
			if i%2 == 0 {
				name = "b"
			}
			if _, err := e.Lookup(context.Background(), gs.Selector{
				Name: name,
			}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("prerequisite initialized", calls.Load(), "times")
	}
}

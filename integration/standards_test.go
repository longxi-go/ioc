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
	"sync/atomic"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	ioccontext "github.com/longxi-go/ioc/pkg/context"
)

func TestOptionalBindingPermitsOnlyAbsence(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		selector   gs.Selector
		candidates []string
		want       gs.Code
	}{
		{
			name: "absent",
			selector: gs.Selector{
				Contract: "unpublished",
			},
		},
		{
			name: "ambiguous",
			selector: gs.Selector{
				Contract: "item",
			},
			candidates: []string{"one", "two"},
			want:       gs.AmbiguousBean,
		},
		{
			name: "named mismatch",
			selector: gs.Selector{
				Contract: "unpublished",
				Name:     "one",
			},
			candidates: []string{"one"},
			want:       gs.ContractMismatch,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			owner := registration("owner")
			owner.Definition.Dependencies = []gs.Dependency{
				{
					Selector:    scenario.selector,
					Access:      gs.Direct,
					Cardinality: gs.ZeroOrOne,
				},
			}
			var called atomic.Int32
			owner.Create = func(ctx context.Context, ports bean.Ports) (any, error) {
				called.Add(1)
				optional, err := bean.OptionalDependency[*item](ctx, ports, 0)
				if err != nil {
					return nil, err
				}
				if optional.Present {
					return nil, errors.New("unexpected optional value")
				}
				return &item{
					name: "owner",
				}, nil
			}
			rs := []bean.Registration{owner}
			for _, name := range scenario.candidates {
				rs = append(rs, registration(name))
			}
			// Exclude the consuming definition from its dependency's contract.
			owner.Definition.Contracts = []gs.Contract{"owner"}
			rs[0] = owner
			app := newApplication(t, rs)
			err := app.Start(context.Background())
			if scenario.want == "" {
				if err != nil || called.Load() != 1 {
					t.Fatalf("absence: calls=%d, err=%v", called.Load(), err)
				}
			} else if !errors.Is(err, &gs.Error{
				Code: scenario.want,
			}) || called.Load() != 0 {
				t.Fatalf("invalid graph must fail before construction: calls=%d, err=%v", called.Load(), err)
			}
		})
	}
}

func TestConditionsSelectBeforeActiveNameValidation(t *testing.T) {
	user, fallback := registration("user"), registration("fallback")
	user.Definition.Name, fallback.Definition.Name = "selected", "selected"
	user.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"implementation"},
				HavingValue: new("user"),
			},
		},
	}
	fallback.Conditions = []bean.Condition{
		{
			Missing: gs.Selector{
				Contract: "item",
			},
		},
	}
	for _, choice := range []string{"user", "default"} {
		t.Run(choice, func(t *testing.T) {
			app := newApplication(t, []bean.Registration{fallback, user}, ioccontext.WithProperties(map[string]string{"implementation": choice}))
			ctx := context.Background()
			if err := app.Start(ctx); err != nil {
				t.Fatal(err)
			}
			value, err := app.GetBean[*item](ctx, "selected")
			want := "fallback"
			if choice == "user" {
				want = "user"
			}
			if err != nil || value == nil || value.name != want {
				t.Fatalf("selected=%v, err=%v", value, err)
			}
		})
	}
}

func TestMutuallyExclusivePropertiesDoNotInventConditionCycle(t *testing.T) {
	a, b := registration("a"), registration("b")
	a.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"mode"},
				HavingValue: new("a"),
			},
			Missing: gs.Selector{
				Name: "b",
			},
		},
	}
	b.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"mode"},
				HavingValue: new("b"),
			},
			Missing: gs.Selector{
				Name: "a",
			},
		},
	}
	app := newApplication(t, []bean.Registration{a, b}, ioccontext.WithProperties(map[string]string{"mode": "a"}))
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("inactive candidate created a condition cycle: %v", err)
	}
	value, err := app.GetBean[*item](ctx)
	if err != nil || value.name != "a" {
		t.Fatalf("active value=%v, error=%v", value, err)
	}
}

func TestQualifiersEmptyCollectionsAndForeignScope(t *testing.T) {
	r := registration("qualified")
	r.Definition.Qualifiers = []string{"blue"}
	app := newApplication(t, []bean.Registration{r})
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Lookup(ctx, gs.Selector{
		Contract:   "item",
		Name:       "qualified",
		Qualifiers: []string{"red"},
	}); !errors.Is(err, &gs.Error{
		Code: gs.MissingBean,
	}) {
		t.Fatal(err)
	}
	values, err := app.LookupAll(ctx, gs.Selector{
		Contract: "absent",
	})
	if err != nil || len(values) != 0 {
		t.Fatalf("empty collection: %v, %v", values, err)
	}
	first := registration("request")
	first.Definition.Scope = gs.Request
	left, right := newApplication(t, []bean.Registration{first}), newApplication(t, []bean.Registration{first})
	for _, a := range []*ioccontext.Application{left, right} {
		if err := a.Start(ctx); err != nil {
			t.Fatal(err)
		}
	}
	scopeCtx, scope, err := left.OpenScope(ctx, gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := scope.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	if _, err := right.GetBean[*item](scopeCtx); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatalf("foreign scope accepted: %v", err)
	}
}

func TestLazyFailureIsStickyAndCompensatesPanics(t *testing.T) {
	r := registration("broken")
	r.Definition.Lazy = true
	var created, destroyed atomic.Int32
	r.Create = func(context.Context, bean.Ports) (any, error) {
		created.Add(1)
		return &item{
			name: "broken",
		}, nil
	}
	r.Init = func(context.Context, any) error {
		panic("failed initialization")
	}
	r.Destroy = func(context.Context, any) error {
		destroyed.Add(1)
		return nil
	}
	app := newApplication(t, []bean.Registration{r})
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if value, err := app.GetBean[*item](ctx); value != nil || !errors.Is(err, &gs.Error{
			Code: gs.CreationFailed,
		}) {
			t.Fatalf("failed bean: %v, %v", value, err)
		}
	}
	if err := app.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if created.Load() != 1 || destroyed.Load() != 1 {
		t.Fatalf("create=%d destroy=%d", created.Load(), destroyed.Load())
	}
}

func TestDecoratorCannotPublishIncompatibleValue(t *testing.T) {
	r := registration("decorated")
	var destroyed atomic.Int32
	r.Destroy = func(context.Context, any) error {
		destroyed.Add(1)
		return nil
	}
	app := newApplication(t, []bean.Registration{r}, ioccontext.WithDecorator(bean.Decorator{
		Contract: "item",
		Wrap: func(context.Context, any) (any, error) {
			return "wrong contract", nil
		},
	}))
	if err := app.Start(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.ContractMismatch,
	}) {
		t.Fatal(err)
	}
	if destroyed.Load() != 1 {
		t.Fatalf("decorated allocation not compensated: %d", destroyed.Load())
	}
}

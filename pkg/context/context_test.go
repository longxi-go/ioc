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

package context_test

import (
	"context"
	"errors"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/ioc/pkg/catalog"
	managed "github.com/longxi-go/ioc/pkg/context"
)

func TestTypedFacade(t *testing.T) {
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:     "b",
		Schema: bean.SchemaVersion,
		Registrations: []bean.Registration{
			{
				Definition: gs.BeanDefinition{
					ID:        "integer",
					Name:      "integer",
					Contracts: []gs.Contract{"int"},
					Scope:     gs.Singleton,
				},
				Create: func(context.Context, bean.Ports) (any, error) {
					return 42, nil
				},
			},
		},
	})
	c.Install(bean.Plan{
		ID:            "app",
		DefinitionIDs: []string{"integer"},
		Query: func(ctx context.Context, f gs.BeanFactory, out any, names []string, all bool) error {
			switch p := out.(type) {
			case *int:
				return bean.Assign(ctx, f, p, gs.Selector{
					Contract: "int",
				})
			case *[]int:
				return bean.AssignAll(ctx, f, p, gs.Selector{
					Contract: "int",
				})
			}
			return &gs.Error{
				Code: gs.UnsupportedContract,
			}
		},
	})
	a, err := managed.NewApplicationContext(context.Background(), managed.WithCatalog(c))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.GetBean[int](context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.InvalidState,
	}) {
		t.Fatal(err)
	}
	if err = a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	v, err := a.GetBean[int](context.Background())
	if err != nil || v != 42 {
		t.Fatal(v, err)
	}
	all, err := a.GetBeans[int](context.Background())
	if err != nil || len(all) != 1 {
		t.Fatal(all, err)
	}
	v, err = a.Provider[int]()(context.Background())
	if err != nil || v != 42 {
		t.Fatal(v, err)
	}
	if err = a.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestTypedNameValidationHonorsLifecycleGate(t *testing.T) {
	a, err := managed.NewApplicationContext(context.Background(), managed.WithCatalog(catalog.New()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.GetBean[int](context.Background(), "one", "two"); !errors.Is(err, &gs.Error{
		Code: gs.InvalidState,
	}) {
		t.Fatal(err)
	}
	if _, err = a.AcquireBean[int](context.Background(), "one", "two"); !errors.Is(err, &gs.Error{
		Code: gs.InvalidState,
	}) {
		t.Fatal(err)
	}
	if err = a.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func testApplication(t *testing.T, r bean.Registration, options ...managed.Option) *managed.Application {
	t.Helper()
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:            "test",
		Schema:        bean.SchemaVersion,
		Registrations: []bean.Registration{r},
	})
	c.Install(bean.Plan{
		ID:            "selected",
		DefinitionIDs: []string{r.Definition.ID},
		Query: func(ctx context.Context, f gs.BeanFactory, out any, names []string, many bool) error {
			s := gs.Selector{
				Contract: "int",
			}
			if len(names) > 0 {
				s.Name = names[0]
			}
			switch p := out.(type) {
			case *int:
				if !many {
					return bean.Assign(ctx, f, p, s)
				}
			case *[]int:
				if many {
					return bean.AssignAll(ctx, f, p, s)
				}
			}
			return &gs.Error{
				Code: gs.UnsupportedContract,
			}
		},
	})
	a, err := managed.NewApplicationContext(context.Background(), append([]managed.Option{managed.WithCatalog(c), managed.WithPlan("selected")}, options...)...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	if err = a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	return a
}

func intRecipe(scope gs.ScopeID) bean.Registration {
	return bean.Registration{
		Definition: gs.BeanDefinition{
			ID:        "integer",
			Name:      "integer",
			Contracts: []gs.Contract{"int"},
			Scope:     scope,
		},
		Create: func(context.Context, bean.Ports) (any, error) {
			return 42, nil
		},
	}
}

func TestFacadeCustomScopeDecorationAndCopiedProperties(t *testing.T) {
	r := intRecipe("tenant")
	r.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"mode"},
				HavingValue: new("on"),
			},
		},
	}
	properties := map[string]string{"mode": "on"}
	option := managed.WithProperties(properties)
	properties["mode"] = "off"
	a := testApplication(t, r, option, managed.WithScope("tenant"), managed.WithDecorator(bean.Decorator{
		Contract: "int",
		Wrap: func(_ context.Context, v any) (any, error) {
			return v.(int) + 1, nil
		},
	}))
	ctx, s, err := a.OpenScope(context.Background(), "tenant")
	if err != nil {
		t.Fatal(err)
	}
	value, err := a.Lookup(ctx, gs.Selector{
		Name: "integer",
	})
	if err != nil || value != 43 {
		t.Fatal(value, err)
	}
	all, err := a.LookupAll(ctx, gs.Selector{
		Contract: "int",
	})
	if err != nil || len(all) != 1 {
		t.Fatal(all, err)
	}
	if a.State() != gs.Running || len(a.Definitions()) != 3 {
		t.Fatal("facade metadata/state")
	}
	if err = s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFacadeResourceLeaseAndAcquisitionErrors(t *testing.T) {
	r := intRecipe(gs.Prototype)
	destroyed := 0
	r.Destroy = func(context.Context, any) error {
		destroyed++
		return nil
	}
	a := testApplication(t, r)
	ctx := context.Background()
	lease, err := a.AcquireBean[int](ctx, "integer")
	if err != nil || lease.Value() != 42 {
		t.Fatal(lease, err)
	}
	if err = lease.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err = lease.Close(ctx); err != nil || destroyed != 1 {
		t.Fatal(destroyed, err)
	}
	if _, err = a.AcquireBean[string](ctx); !errors.Is(err, &gs.Error{
		Code: gs.UnsupportedContract,
	}) {
		t.Fatal(err)
	}
	if _, err = a.AcquireBean[int](ctx, "a", "b"); !errors.Is(err, &gs.Error{
		Code: gs.InvalidSelector,
	}) {
		t.Fatal(err)
	}
	if _, err = a.GetBean[int](ctx, "a", "b"); !errors.Is(err, &gs.Error{
		Code: gs.InvalidSelector,
	}) {
		t.Fatal(err)
	}
	if _, err = a.AcquireBean[int](ctx, "missing"); !errors.Is(err, &gs.Error{
		Code: gs.MissingBean,
	}) {
		t.Fatal(err)
	}
}

func TestFacadeEmptyUnsupportedAndInvalidConfiguration(t *testing.T) {
	if _, err := managed.NewApplicationContext(context.Background(), managed.WithCatalog(nil)); err == nil {
		t.Fatal("nil catalog")
	}
	c := catalog.New()
	c.Install(bean.Plan{
		ID:            "broken",
		DefinitionIDs: []string{"absent"},
	})
	if _, err := managed.NewApplicationContext(context.Background(), managed.WithCatalog(c)); err == nil {
		t.Fatal("invalid plan")
	}
	a, err := managed.NewApplicationContext(context.Background(), managed.WithCatalog(catalog.New()), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = a.GetBean[int](context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.UnsupportedContract,
	}) {
		t.Fatal(err)
	}
	if err = a.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

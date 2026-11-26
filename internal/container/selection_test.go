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
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func TestSelectionCardinalityNamesAndPrimaries(t *testing.T) {
	a := constant("a")
	b := constant("b")
	a.Definition.Contracts = []gs.Contract{"service"}
	b.Definition.Contracts = []gs.Contract{"service"}
	a.Definition.Aliases = []string{"alias"}
	a.Definition.Qualifiers = []string{"q"}
	b.Definition.Primary = true
	b.Definition.Order = -1
	e := boot(t, a, b)
	one, err := e.Lookup(context.Background(), gs.Selector{
		Contract: "service",
	})
	if err != nil {
		t.Fatal(err)
	}
	named, err := e.Lookup(context.Background(), gs.Selector{
		Name: "b",
	})
	if err != nil || one != named {
		t.Fatal(err)
	}
	all, err := e.LookupAll(context.Background(), gs.Selector{
		Contract: "service",
	})
	if err != nil || len(all) != 2 || all[0] != one {
		t.Fatal(all, err)
	}
	for _, s := range []gs.Selector{
		{
			Name:     "alias",
			Contract: "wrong",
		},
		{
			Name:       "alias",
			Contract:   "wrong",
			Qualifiers: []string{"absent"},
		},
	} {
		if _, err = e.Lookup(context.Background(), s); !errors.Is(err, &gs.Error{
			Code: gs.ContractMismatch,
		}) {
			t.Fatal(err)
		}
	}
	if _, err = e.Lookup(context.Background(), gs.Selector{
		Name:       "alias",
		Qualifiers: []string{"absent"},
	}); !errors.Is(err, &gs.Error{
		Code: gs.MissingBean,
	}) {
		t.Fatal(err)
	}
	if _, err = e.Lookup(context.Background(), gs.Selector{}); !errors.Is(err, &gs.Error{
		Code: gs.InvalidSelector,
	}) {
		t.Fatal(err)
	}
	if all, err = e.LookupAll(context.Background(), gs.Selector{
		Contract: "absent",
	}); err != nil || len(all) != 0 {
		t.Fatal(all, err)
	}
}

func TestConditionsObserveAcceptedUserCandidates(t *testing.T) {
	user := constant("user")
	user.Definition.Contracts = []gs.Contract{"service"}
	user.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"enabled"},
				HavingValue: conditionString("yes"),
			},
		},
	}
	fallback := constant("fallback")
	fallback.Definition.Contracts = []gs.Contract{"service"}
	fallback.Conditions = []bean.Condition{
		{
			Missing: gs.Selector{
				Contract: "service",
			},
		},
	}
	dependent := constant("dependent")
	dependent.Conditions = []bean.Condition{
		{
			Present: gs.Selector{
				Name: "fallback",
			},
		},
	}
	for _, enabled := range []string{"yes", "no"} {
		t.Run(enabled, func(t *testing.T) {
			e, err := New([]bean.Registration{fallback, user, dependent}, Options{
				Snapshot: conditionSnapshot(t, map[string]string{"enabled": enabled}),
			})
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err = e.Close(context.Background()); err != nil {
					t.Error(err)
				}
			}()
			name := "user"
			if enabled == "no" {
				name = "fallback"
			}
			v, err := e.Lookup(context.Background(), gs.Selector{
				Contract: "service",
			})
			if err != nil {
				t.Fatal(err)
			}
			expected, err := e.Lookup(context.Background(), gs.Selector{
				Name: name,
			})
			if err != nil || v != expected {
				t.Fatal(err)
			}
			_, err = e.Lookup(context.Background(), gs.Selector{
				Name: "dependent",
			})
			if enabled == "no" && err != nil || enabled == "yes" && !errors.Is(err, &gs.Error{
				Code: gs.MissingBean,
			}) {
				t.Fatal(err)
			}
		})
	}
}

func TestReciprocalMissingCannotBeResolvedByInitializationOrder(t *testing.T) {
	a := constant("a")
	b := constant("b")
	a.Conditions = []bean.Condition{
		{
			Missing: gs.Selector{
				Name: "b",
			},
		},
	}
	b.Conditions = []bean.Condition{
		{
			Missing: gs.Selector{
				Name: "a",
			},
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
	a.Definition.DependsOn = []string{"b"}
	e, err = New([]bean.Registration{a, b}, Options{})
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

func TestOptionalDoesNotSwallowAmbiguity(t *testing.T) {
	a := constant("a")
	b := constant("b")
	a.Definition.Contracts = []gs.Contract{"service"}
	b.Definition.Contracts = []gs.Contract{"service"}
	consumer := constant("consumer")
	consumer.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Contract: "service",
			},
			Access:      gs.Direct,
			Cardinality: gs.ZeroOrOne,
		},
	}
	e, err := New([]bean.Registration{consumer, a, b}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.AmbiguousBean,
	}) {
		t.Fatal(err)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDeferredPortUsesInvocationScope(t *testing.T) {
	r := recipe("request", gs.Request, func(context.Context, bean.Ports) (any, error) {
		return new(int), nil
	})
	consumer := recipe("consumer", gs.Singleton, func(_ context.Context, p bean.Ports) (any, error) {
		return gs.Provider[*int](func(ctx context.Context) (*int, error) {
			return bean.Dependency[*int](ctx, p, 0)
		}), nil
	})
	consumer.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "request",
			},
			Access:      gs.Deferred,
			Cardinality: gs.One,
		},
	}
	e := boot(t, r, consumer)
	v, err := e.Lookup(context.Background(), gs.Selector{
		Name: "consumer",
	})
	if err != nil {
		t.Fatal(err)
	}
	provider := v.(gs.Provider[*int])
	if _, err = provider(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
	ctx, s, err := e.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = s.Close(context.Background()); err != nil {
			t.Error(err)
		}
	}()
	a, err := provider(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err := provider(ctx)
	if err != nil || a != b {
		t.Fatal(err)
	}
}

func TestCycleDiagnosticIncludesPathAndDependencySource(t *testing.T) {
	a := constant("a")
	b := constant("b")
	a.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "b",
			},
			Access:      gs.Direct,
			Cardinality: gs.One,
			Source: gs.Source{
				File: "wiring.go",
				Line: 11,
			},
		},
	}
	b.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "a",
			},
			Access:      gs.Direct,
			Cardinality: gs.One,
			Source: gs.Source{
				File: "wiring.go",
				Line: 17,
			},
		},
	}
	e, err := New([]bean.Registration{a, b}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	err = e.Start(context.Background())
	var diagnostic *gs.Error
	if !errors.As(err, &diagnostic) ||
		len(diagnostic.Path) < 3 ||
		diagnostic.Source.File != "wiring.go" ||
		diagnostic.Bean == "" {
		t.Fatalf("missing diagnostic context: %#v", diagnostic)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPrototypeCannotHideTransitiveScopedCapture(t *testing.T) {
	r := constant("request")
	r.Definition.Scope = gs.Request
	p := constant("prototype")
	p.Definition.Scope = gs.Prototype
	p.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "tenant",
			},
			Access:      gs.Direct,
			Cardinality: gs.One,
		},
	}
	tenant := constant("tenant")
	tenant.Definition.Scope = "tenant"
	r.Definition.Dependencies = []gs.Dependency{
		{
			Selector: gs.Selector{
				Name: "prototype",
			},
			Access:      gs.Direct,
			Cardinality: gs.One,
		},
	}
	e, err := New([]bean.Registration{r, p, tenant}, Options{
		Scopes: []gs.ScopeID{"tenant"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Start(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal("transitive capture accepted", err)
	}
	if err = e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSingletonReceivesFreshOrdinaryPrototypePerConsumer(t *testing.T) {
	p := recipe("p", gs.Prototype, func(context.Context, bean.Ports) (any, error) {
		return new(int), nil
	})
	makeConsumer := func(id string) bean.Registration {
		r := recipe(id, gs.Singleton, func(ctx context.Context, ports bean.Ports) (any, error) {
			return bean.Dependency[*int](ctx, ports, 0)
		})
		r.Definition.Dependencies = []gs.Dependency{
			{
				Selector: gs.Selector{
					Name: "p",
				},
				Access:      gs.Direct,
				Cardinality: gs.One,
			},
		}
		return r
	}
	e := boot(t, p, makeConsumer("a"), makeConsumer("b"))
	a, err := e.Lookup(context.Background(), gs.Selector{
		Name: "a",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Lookup(context.Background(), gs.Selector{
		Name: "b",
	})
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := e.Lookup(context.Background(), gs.Selector{
		Name: "p",
	})
	if err != nil ||
		a == b ||
		fresh == a ||
		fresh == b {
		t.Fatal("prototype reused across consumers", err)
	}
}

func TestPropertyIneligibleCandidatesCannotCreateConditionOrOrderCycles(t *testing.T) {
	for _, ordered := range []bool{false, true} {
		t.Run(fmt.Sprint(ordered), func(t *testing.T) {
			a := constant("a")
			b := constant("b")
			a.Conditions = []bean.Condition{
				{
					Property: &bean.PropertyCondition{
						Names:       []string{"mode"},
						HavingValue: conditionString("a"),
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
						HavingValue: conditionString("b"),
					},
					Missing: gs.Selector{
						Name: "a",
					},
				},
			}
			if ordered {
				// Only the excluded definition requires the active one. An
				// excluded dependent creates no prerequisite requirement.
				b.Definition.DependsOn = []string{"a"}
			}
			e, err := New([]bean.Registration{a, b}, Options{
				Snapshot: conditionSnapshot(t, map[string]string{"mode": "a"}),
			})
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Start(context.Background()); err != nil {
				t.Fatal("ineligible property candidate invented a cycle", err)
			}
			if _, err = e.Lookup(context.Background(), gs.Selector{
				Name: "a",
			}); err != nil {
				t.Fatal(err)
			}
			if _, err = e.Lookup(context.Background(), gs.Selector{
				Name: "b",
			}); !errors.Is(err, &gs.Error{
				Code: gs.MissingBean,
			}) {
				t.Fatal(err)
			}
			if err = e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

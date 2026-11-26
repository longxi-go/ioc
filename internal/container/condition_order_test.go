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
	"slices"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func TestCollectionOrderDoesNotOrderInitialization(t *testing.T) {
	for _, order := range []int{-100, 100} {
		var trace []string
		a, b := constant("a"), constant("b")
		a.Definition.Order = order
		for _, r := range []*bean.Registration{&a, &b} {
			id := r.Definition.ID
			r.Definition.Contracts = []gs.Contract{"value"}
			r.Init = func(context.Context, any) error {
				trace = append(trace, id)
				return nil
			}
		}
		e := boot(t, b, a)
		if !slices.Equal(trace, []string{"a", "b"}) {
			t.Fatalf("collection Order=%d changed independent initialization: %v", order, trace)
		}
		matches, err := choose(e.bound, gs.Selector{
			Contract: a.Definition.Contracts[0],
		}, true, false)
		if err != nil || len(matches) != 2 {
			t.Fatal(matches, err)
		}
		want := "a"
		if order > 0 {
			want = "b"
		}
		if matches[0].recipe.Definition.ID != want {
			t.Fatalf("collection Order=%d not respected: %v", order, matches)
		}
	}
}

func TestConditionSelectionIndependentOfOrder(t *testing.T) {
	for _, order := range []int{-100, 0, 100} {
		user, fallback := constant("user"), constant("fallback")
		user.Definition.Order = order
		user.Definition.Contracts = []gs.Contract{"repository"}
		fallback.Definition.Contracts = []gs.Contract{"repository"}
		fallback.Conditions = []bean.Condition{
			{
				Missing: gs.Selector{
					Contract: "repository",
				},
			},
		}
		for _, rs := range [][]bean.Registration{
			{
				fallback,
				user,
			},
			{
				user,
				fallback,
			},
		} {
			accepted, err := selectRecipes(rs, nil)
			if err != nil || len(accepted) != 1 || accepted[0].Definition.ID != "user" {
				t.Fatalf("order=%d, accepted=%v err=%v", order, accepted, err)
			}
		}
	}
}

func TestConditionSelectionIndependentOfPrerequisites(t *testing.T) {
	user, fallback := constant("user"), constant("fallback")
	user.Definition.Contracts = []gs.Contract{"repository"}
	fallback.Conditions = []bean.Condition{
		{
			Missing: gs.Selector{
				Contract: "repository",
			},
		},
	}
	user.Definition.DependsOn = []string{"fallback"}
	accepted, err := selectRecipes([]bean.Registration{fallback, user}, nil)
	if err != nil || len(accepted) != 1 || accepted[0].Definition.ID != "user" {
		t.Fatal(accepted, err)
	}
}

func TestConditionalCyclesDoNotUsePrimary(t *testing.T) {
	for _, positive := range []bool{false, true} {
		for _, primary := range []bool{false, true} {
			a, b := constant("a"), constant("b")
			a.Definition.Primary = primary
			a.Definition.Order = -100
			ca, cb := bean.Condition{
				Missing: gs.Selector{
					Name: "b",
				},
			}, bean.Condition{
				Missing: gs.Selector{
					Name: "a",
				},
			}
			if positive {
				ca = bean.Condition{
					Present: gs.Selector{
						Name: "b",
					},
				}
			}
			a.Conditions, b.Conditions = []bean.Condition{ca}, []bean.Condition{cb}
			_, err := selectRecipes([]bean.Registration{a, b}, nil)
			var detail *gs.Error
			if !errors.As(err, &detail) || detail.Code != gs.DependencyCycle || len(detail.Path) < 3 {
				t.Fatalf("positive=%v primary=%v: %v", positive, primary, err)
			}
		}
	}
}

func TestConditionSelectionIndependentOfInputPermutation(t *testing.T) {
	a, b, c := constant("a"), constant("b"), constant("c")
	b.Conditions = []bean.Condition{
		{
			Present: gs.Selector{
				Name: "a",
			},
		},
	}
	c.Conditions = []bean.Condition{
		{
			Missing: gs.Selector{
				Name: "b",
			},
		},
	}
	for _, rs := range [][]bean.Registration{
		{
			a,
			b,
			c,
		},
		{
			c,
			b,
			a,
		},
		{
			b,
			c,
			a,
		},
	} {
		accepted, err := selectRecipes(rs, nil)
		var ids []string
		for _, r := range accepted {
			ids = append(ids, r.Definition.ID)
		}
		if err != nil || !slices.Equal(ids, []string{"a", "b"}) {
			t.Fatal(ids, err)
		}
	}
}

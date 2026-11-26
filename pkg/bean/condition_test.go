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

package bean_test

import (
	"context"
	"errors"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func unusedConditionFactory(context.Context, bean.Ports) (any, error) {
	panic("metadata validation must not construct values")
}

func TestRegistrationValidatesConditionPredicates(t *testing.T) {
	cases := []struct {
		name      string
		condition bean.Condition
		valid     bool
	}{
		{
			"empty predicate",
			bean.Condition{},
			false,
		},
		{
			"value without key",
			bean.Condition{
				Property: &bean.PropertyCondition{},
			},
			false,
		},
		{
			"blank key",
			bean.Condition{
				Property: &bean.PropertyCondition{
					Names: []string{" "},
				},
			},
			false,
		},
		{
			"empty compared value",
			bean.Condition{
				Property: &bean.PropertyCondition{
					Names: []string{"mode"},
				},
			},
			true,
		},
		{
			"missing selector",
			bean.Condition{
				Missing: gs.Selector{
					Name: "fallback",
				},
			},
			true,
		},
		{
			"present selector",
			bean.Condition{
				Present: gs.Selector{
					Contract: "Service",
				},
			},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := bean.Registration{
				Definition: gs.BeanDefinition{
					ID:        "test/condition",
					Name:      "condition",
					Scope:     gs.Singleton,
					Contracts: []gs.Contract{"bool"},
				},
				Create:     unusedConditionFactory,
				Conditions: []bean.Condition{tc.condition},
			}
			err := r.Validate()
			if tc.valid && err != nil {
				t.Fatal(err)
			}
			if !tc.valid && !errors.Is(err, &gs.Error{
				Code: gs.InvalidDefinition,
			}) {
				t.Fatalf("invalid predicate accepted: %v", err)
			}
		})
	}
}

func TestPropertyAndProfileMetadata(t *testing.T) {
	for _, tc := range []struct {
		p     bean.PropertyCondition
		valid bool
	}{
		{
			bean.PropertyCondition{},
			false,
		},
		{
			bean.PropertyCondition{
				Names: []string{"x", "x"},
			},
			false,
		},
		{
			bean.PropertyCondition{
				Prefix: " ",
				Names:  []string{"x"},
			},
			false,
		},
		{
			bean.PropertyCondition{
				Prefix: "app..",
				Names:  []string{"x"},
			},
			false,
		},
		{
			bean.PropertyCondition{
				Prefix: "app.",
				Names:  []string{"x"},
			},
			true,
		},
		{
			bean.PropertyCondition{
				Names: []string{"${key}"},
			},
			true,
		},
		{
			bean.PropertyCondition{
				Names: []string{" x"},
			},
			false,
		},
	} {
		if err := (bean.Condition{
			Property: &tc.p,
		}).Validate(); (err == nil) != tc.valid {
			t.Errorf("%+v: %v", tc.p, err)
		}
	}
	for _, profile := range []string{"prod", "!dev"} {
		if err := (bean.Condition{
			Profiles: []string{profile},
		}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, profile := range []string{"", "!", "!!dev", " dev"} {
		if err := (bean.Condition{
			Profiles: []string{profile},
		}).Validate(); err == nil {
			t.Errorf("accepted invalid profile %q", profile)
		}
	}
}

func TestConditionCloneOwnsMetadata(t *testing.T) {
	value := ""
	original := bean.Registration{
		Conditions: []bean.Condition{
			{
				Property: &bean.PropertyCondition{
					Names:       []string{"x"},
					HavingValue: &value,
				},
				Profiles: []string{"prod"},
				Missing: gs.Selector{
					Qualifiers: []string{"one"},
				},
				Present: gs.Selector{
					Qualifiers: []string{"two"},
				},
			},
		},
	}
	cloned := original.Clone()
	cloned.Conditions[0].Property.Names[0] = "y"
	*cloned.Conditions[0].Property.HavingValue = "changed"
	cloned.Conditions[0].Profiles[0] = "dev"
	cloned.Conditions[0].Missing.Qualifiers[0] = "changed"
	cloned.Conditions[0].Present.Qualifiers[0] = "changed"
	c := original.Conditions[0]
	if c.Property.Names[0] != "x" ||
		*c.Property.HavingValue != "" ||
		c.Profiles[0] != "prod" ||
		c.Missing.Qualifiers[0] != "one" ||
		c.Present.Qualifiers[0] != "two" {
		t.Fatal("clone aliases original metadata")
	}
}

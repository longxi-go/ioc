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
	"strings"
	"testing"

	"github.com/longxi-go/gosr/pkg/environment"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/nemo"
	options "github.com/longxi-go/nemo/pkg/environment"
	"github.com/longxi-go/nemo/pkg/maptypes"
	"github.com/longxi-go/nemo/pkg/source"
)

func conditionString(s string) *string {
	return &s
}

func conditionSnapshot(t *testing.T, values map[string]string) environment.Snapshot {
	t.Helper()
	raw := maptypes.ValueMap{}
	for k, v := range values {
		raw[k] = environment.String(v)
	}
	src, err := source.NewMap("conditions", raw)
	if err != nil {
		t.Fatal(err)
	}
	env, err := nemo.New(context.Background(), options.WithProviders(source.Static(src)))
	if err != nil {
		t.Fatal(err)
	}
	return env.Snapshot()
}

func TestTypedPropertyConditions(t *testing.T) {
	for _, tc := range []struct {
		name, value    string
		expected       *string
		missing, match bool
	}{
		{
			name:  "presence",
			value: "yes",
			match: true,
		},
		{
			name:  "false",
			value: "FaLsE",
		},
		{
			name:     "empty",
			expected: conditionString(""),
			match:    true,
		},
		{
			name:     "case",
			value:    "YES",
			expected: conditionString("yes"),
		},
		{
			name:    "missing",
			missing: true,
			match:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := constant("candidate")
			r.Conditions = []bean.Condition{
				{
					Property: &bean.PropertyCondition{
						Prefix:         "app.",
						Names:          []string{"enabled"},
						HavingValue:    tc.expected,
						MatchIfMissing: tc.missing,
					},
				},
			}
			values := map[string]string{}
			if !tc.missing {
				values["app.enabled"] = tc.value
			}
			result, err := evaluateProperties(r, conditionSnapshot(t, values))
			if err != nil {
				t.Fatal(err)
			}
			if got := result.Outcome != bean.Rejected; got != tc.match {
				t.Fatalf("matched %v", got)
			}
		})
	}
}

func TestProfileCondition(t *testing.T) {
	r := constant("candidate")
	r.Conditions = []bean.Condition{
		{
			Profiles: []string{"absent", "default"},
		},
	}
	result, err := evaluateProperties(r, conditionSnapshot(t, nil))
	if err != nil || result.Outcome == bean.Rejected {
		t.Fatalf("%+v %v", result, err)
	}
}

type failingConditionSnapshot struct {
	environment.Snapshot
	cause error
}

func (s failingConditionSnapshot) Lookup(string) (environment.Property, bool, error) {
	return environment.Property{}, false, s.cause
}

func TestConditionFailureRedactsCause(t *testing.T) {
	cause := errors.New("private-secret")
	r := constant("candidate")
	r.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names: []string{"key"},
			},
		},
	}
	_, err := evaluateProperties(r, failingConditionSnapshot{
		Snapshot: conditionSnapshot(t, nil),
		cause:    cause,
	})
	if err == nil || !errors.Is(err, cause) || strings.Contains(err.Error(), "private-secret") {
		t.Fatalf("unsafe or lost cause: %v", err)
	}
}

func TestPropertyConditionRequiresSnapshot(t *testing.T) {
	r := constant("candidate")
	r.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names: []string{"key"},
			},
		},
	}
	_, err := evaluateProperties(r, nil)
	if !errors.Is(err, &gs.Error{
		Code: gs.InvalidDefinition,
	}) {
		t.Fatalf("expected classified failure: %v", err)
	}
}

func TestResolvedConditionAddresses(t *testing.T) {
	for _, prefix := range []string{"private.", ""} {
		t.Run(prefix, func(t *testing.T) {
			key := "enabled"
			if prefix != "" {
				key = "private.enabled"
			}
			snapshot := conditionSnapshot(t, map[string]string{"prefix": prefix, "name": "enabled", key: "yes"})
			r := constant("candidate")
			r.Conditions = []bean.Condition{
				{
					Property: &bean.PropertyCondition{
						Prefix: "${prefix}",
						Names:  []string{"${name}"},
					},
				},
			}
			result, err := evaluateProperties(r, snapshot)
			if err != nil || result.Outcome == bean.Rejected {
				t.Fatalf("%+v %v", result, err)
			}
			if result.Predicates[0].Property != "${prefix}.${name}" {
				t.Fatalf("resolved operand leaked: %+v", result)
			}
		})
	}
}

func TestPropertyValueKindsAndMissing(t *testing.T) {
	number, err := environment.Number("42")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                               string
		value                              environment.Value
		present, missing, matched, invalid bool
	}{
		{
			name:    "boolean",
			value:   environment.Bool(true),
			present: true,
			matched: true,
		},
		{
			name:    "false",
			value:   environment.Bool(false),
			present: true,
		},
		{
			name:    "number",
			value:   number,
			present: true,
			matched: true,
		},
		{
			name:    "null",
			present: true,
			invalid: true,
		},
		{
			name:    "list",
			value:   environment.List(),
			present: true,
			invalid: true,
		},
		{
			name:    "object",
			value:   environment.Object(nil),
			present: true,
			invalid: true,
		},
		{
			name: "missing rejects",
		},
		{
			name:    "missing matches",
			missing: true,
			matched: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := maptypes.ValueMap{}
			if tc.present {
				raw["key"] = tc.value
			}
			src, err := source.NewMap("values", raw)
			if err != nil {
				t.Fatal(err)
			}
			owner, err := nemo.New(context.Background(), options.WithProviders(source.Static(src)))
			if err != nil {
				t.Fatal(err)
			}
			r := constant("candidate")
			r.Conditions = []bean.Condition{
				{
					Property: &bean.PropertyCondition{
						Names:          []string{"key"},
						MatchIfMissing: tc.missing,
					},
				},
			}
			result, err := evaluateProperties(r, owner.Snapshot())
			if tc.invalid {
				if !errors.Is(err, &gs.Error{
					Code: gs.InvalidDefinition,
				}) {
					t.Fatalf("%v", err)
				}
				return
			}
			if err != nil || (result.Outcome != bean.Rejected) != tc.matched {
				t.Fatalf("%+v %v", result, err)
			}
		})
	}
}

func TestPropertyNamesAndRepeatedConditionsAreAND(t *testing.T) {
	r := constant("candidate")
	r.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names: []string{"a", "b"},
			},
		},
		{
			Property: &bean.PropertyCondition{
				Names: []string{"c"},
			},
		},
	}
	for _, values := range []map[string]string{
		{
			"a": "yes",
			"b": "false",
			"c": "yes",
		},
		{
			"a": "yes",
			"b": "yes",
			"c": "false",
		},
	} {
		result, err := evaluateProperties(r, conditionSnapshot(t, values))
		if err != nil || result.Outcome != bean.Rejected || len(result.Predicates) != 3 {
			t.Fatalf("%+v %v", result, err)
		}
	}
}

func TestConditionOperandsResolveOnce(t *testing.T) {
	r := constant("candidate")
	r.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{`${${selector}}`},
				HavingValue: conditionString(`\${literal}`),
			},
		},
	}
	result, err := evaluateProperties(r, conditionSnapshot(t, map[string]string{"selector": "address", "address": "enabled", "enabled": `\${literal}`}))
	if err != nil || result.Outcome == bean.Rejected {
		t.Fatalf("%+v %v", result, err)
	}
	// Escaped operands produce literal keys rather than a second interpolation pass.
	r.Conditions[0].Property.Names = []string{`\${literal}`}
	result, err = evaluateProperties(r, conditionSnapshot(t, map[string]string{"${literal}": `\${literal}`}))
	if err != nil || result.Outcome == bean.Rejected {
		t.Fatalf("%+v %v", result, err)
	}
}

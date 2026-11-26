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
	"reflect"
	"sync"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func TestConditionReportMatchesStartupWithoutConstructing(t *testing.T) {
	calls := 0
	user := recipe("user", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
		calls++
		return new(int), nil
	})
	user.Definition.Contracts = []gs.Contract{"service"}
	user.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"password"},
				HavingValue: conditionString("secret"),
			},
		},
	}
	fallback := constant("fallback")
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
				Contract:   "service",
				Qualifiers: []string{"q"},
			},
		},
	}
	user.Definition.Qualifiers = []string{"q"}
	for _, enabled := range []bool{true, false} {
		t.Run(map[bool]string{true: "accepted", false: "property-filtered"}[enabled], func(t *testing.T) {
			properties := map[string]string{}
			if enabled {
				properties["password"] = "secret"
			}
			e, err := New([]bean.Registration{fallback, dependent, user}, Options{
				Snapshot: conditionSnapshot(t, properties),
			})
			if err != nil {
				t.Fatal(err)
			}
			beforeCalls := calls
			report, err := e.ConditionReport()
			if err != nil || calls != beforeCalls || e.State() != gs.Created {
				t.Fatalf("report executed user code: %v, %d", err, calls)
			}
			outcomes := map[string]bean.CandidateEvaluation{}
			for _, candidate := range report.Candidates {
				outcomes[candidate.Bean] = candidate
			}
			wantUser, wantFallback := bean.Accepted, bean.Rejected
			if !enabled {
				wantUser, wantFallback = bean.Rejected, bean.Accepted
			}
			if outcomes["user"].Outcome != wantUser ||
				outcomes["fallback"].Outcome != wantFallback ||
				outcomes["dependent"].Outcome != wantUser {
				t.Fatal(outcomes)
			}
			if enabled && !reflect.DeepEqual(outcomes["dependent"].Predicates[0].Matches, []string{"user"}) {
				t.Fatal(outcomes["dependent"])
			}
			if !enabled && outcomes["user"].Predicates[0].Matched {
				t.Fatal("property filter reported success")
			}
			if err := e.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			for _, definition := range e.Definitions() {
				if outcomes[definition.ID].Outcome != bean.Accepted {
					t.Fatalf("report disagrees with Start for %s", definition.ID)
				}
			}
			again, err := e.ConditionReport()
			if err != nil || !reflect.DeepEqual(report, again) {
				t.Fatal("report changed after Start", err)
			}
			for i := range report.Candidates {
				for j := range report.Candidates[i].Predicates {
					p := &report.Candidates[i].Predicates[j]
					if len(p.Matches) > 0 {
						p.Matches[0] = "mutated"
					}
					if len(p.Selector.Qualifiers) > 0 {
						p.Selector.Qualifiers[0] = "mutated"
					}
				}
			}
			fresh, err := e.ConditionReport()
			if err != nil || !reflect.DeepEqual(again, fresh) {
				t.Fatal("report aliases application metadata", err)
			}
			if err := e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConditionReportPreservesUnevaluatedCycles(t *testing.T) {
	a, b, excluded := constant("a"), constant("b"), constant("excluded")
	a.Conditions = []bean.Condition{
		{
			Present: gs.Selector{
				Name: "b",
			},
		},
	}
	b.Conditions = []bean.Condition{
		{
			Present: gs.Selector{
				Name: "a",
			},
		},
	}
	excluded.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"enabled"},
				HavingValue: conditionString("yes"),
			},
		},
	}
	e, err := New([]bean.Registration{b, excluded, a}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := e.ConditionReport()
	if !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatal(err)
	}
	for _, c := range report.Candidates {
		if c.Bean == "excluded" {
			if c.Outcome != bean.Rejected {
				t.Fatal(c)
			}
		} else if c.Outcome != bean.NotEvaluated {
			t.Fatal(c)
		}
	}
	if err := e.Start(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatal(err)
	}
	if err := e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestConditionReportsAreConcurrentWithLifecycle(t *testing.T) {
	e, err := New([]bean.Registration{constant("value")}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 30 {
				if _, err := e.ConditionReport(); err != nil {
					t.Error(err)
				}
			}
		})
	}
	if err := e.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := e.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

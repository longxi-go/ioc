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

package catalog_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/ioc/pkg/catalog"
)

func TestOldGeneratedBatchRequiresRegeneration(t *testing.T) {
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:            "legacy",
		Schema:        2,
		Registrations: []bean.Registration{reg("legacy")},
	})
	_, recipes, err := c.Snapshot("")
	if !errors.Is(err, &gs.Error{
		Code: gs.InvalidDefinition,
	}) || !strings.Contains(err.Error(), "regenerate") {
		t.Fatalf("old generated ABI must require regeneration: %v", err)
	}
	if len(recipes) != 0 {
		t.Fatal("incompatible recipes were published", recipes)
	}
}

func TestSnapshotOwnsListenerOrderAcrossBothCopyBoundaries(t *testing.T) {
	order := 100
	r := reg("audit")
	r.Listeners = []bean.Listener{
		{
			ID:    "audit.OnEvent",
			Event: "Event",
			Order: &order,
			Match: func(event.Event) bool {
				return true
			},
			Invoke: func(context.Context, any, event.Event) error {
				return nil
			},
		},
	}
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:            "events",
		Schema:        bean.SchemaVersion,
		Registrations: []bean.Registration{r},
	})
	order = -100
	r.Listeners[0].ID = "changed"
	_, first, err := c.Snapshot("")
	if err != nil || first[0].Listeners[0].ID != "audit.OnEvent" || *first[0].Listeners[0].Order != 100 {
		t.Fatalf("contribution ownership: %v %v", first, err)
	}
	*first[0].Listeners[0].Order = 200
	_, second, err := c.Snapshot("")
	if err != nil || *second[0].Listeners[0].Order != 100 {
		t.Fatalf("snapshot ownership: %v %v", second, err)
	}
}

func reg(id string) bean.Registration {
	return bean.Registration{
		Definition: gs.BeanDefinition{
			ID:        id,
			Name:      id,
			Contracts: []gs.Contract{"int"},
			Scope:     gs.Singleton,
		},
		Create: func(context.Context, bean.Ports) (any, error) {
			return 1, nil
		},
	}
}

func TestAtomicOwnedSnapshot(t *testing.T) {
	c := catalog.New()
	r := reg("one")
	c.Contribute(bean.Batch{
		ID:            "batch",
		Schema:        bean.SchemaVersion,
		Registrations: []bean.Registration{r},
	})
	r.Definition.Contracts[0] = "changed"
	_, rs, err := c.Snapshot("")
	if err != nil || len(rs) != 1 || rs[0].Definition.Contracts[0] != "int" {
		t.Fatal(rs, err)
	}
	rs[0].Definition.Contracts[0] = "changed"
	_, rs, err = c.Snapshot("")
	if err != nil || rs[0].Definition.Contracts[0] != "int" {
		t.Fatal(rs, err)
	}
	c.Contribute(bean.Batch{
		ID:            "bad",
		Schema:        bean.SchemaVersion,
		Registrations: []bean.Registration{reg("two"), reg("one")},
	})
	if _, _, err = c.Snapshot(""); err == nil {
		t.Fatal("invalid contribution not recorded")
	}
}

func TestPlanClosure(t *testing.T) {
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:            "b",
		Schema:        bean.SchemaVersion,
		Registrations: []bean.Registration{reg("one"), reg("two")},
	})
	p := bean.Plan{
		ID:            "app",
		DefinitionIDs: []string{"two"},
	}
	c.Install(p)
	p.DefinitionIDs[0] = "bad"
	_, rs, err := c.Snapshot("app")
	if err != nil || len(rs) != 1 || rs[0].Definition.ID != "two" {
		t.Fatal(rs, err)
	}
}

func TestContributionRejectsInvalidWithoutCallbacks(t *testing.T) {
	for _, name := range []string{"identity", "schema", "metadata", "factory", "duplicate", "condition"} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			r := reg("a")
			r.Create = func(context.Context, bean.Ports) (any, error) {
				calls++
				return 1, nil
			}
			batch := bean.Batch{
				ID:            "batch",
				Schema:        bean.SchemaVersion,
				Registrations: []bean.Registration{r},
			}
			switch name {
			case "identity":
				batch.ID = ""
			case "schema":
				batch.Schema++
			case "metadata":
				batch.Registrations[0].Definition.Scope = ""
			case "factory":
				batch.Registrations[0].Create = nil
			case "duplicate":
				batch.Registrations = append(batch.Registrations, r)
			case "condition":
				batch.Registrations[0].Conditions = []bean.Condition{
					{
						Missing: gs.Selector{
							Contract: " ",
						},
					},
				}
			}
			c := catalog.New()
			c.Contribute(batch)
			if _, _, err := c.Snapshot(""); err == nil || calls != 0 {
				t.Fatal(calls, err)
			}
		})
	}
}

func TestPlanValidationAndDefaultSelection(t *testing.T) {
	for _, scenario := range []string{"empty", "duplicate", "unknown", "missing", "repeated", "multiple"} {
		t.Run(scenario, func(t *testing.T) {
			c := catalog.New()
			c.Contribute(bean.Batch{
				ID:            "batch",
				Schema:        bean.SchemaVersion,
				Registrations: []bean.Registration{reg("a")},
			})
			plan := bean.Plan{
				ID:            "app",
				DefinitionIDs: []string{"a"},
			}
			query := ""
			switch scenario {
			case "empty":
				plan.ID = ""
			case "duplicate":
				c.Install(plan)
			case "unknown":
				query = "unknown"
			case "missing":
				plan.DefinitionIDs = []string{"missing"}
			case "repeated":
				plan.DefinitionIDs = []string{"a", "a"}
			case "multiple":
				c.Install(bean.Plan{
					ID:            "other",
					DefinitionIDs: []string{"a"},
				})
			}
			c.Install(plan)
			if _, _, err := c.Snapshot(query); err == nil {
				t.Fatal("invalid plan accepted")
			}
		})
	}
	c := catalog.New()
	c.Install(bean.Plan{
		ID:            "empty",
		DefinitionIDs: []string{},
	})
	plan, rs, err := c.Snapshot("")
	if err != nil || plan.ID != "empty" || len(rs) != 0 {
		t.Fatal(plan, rs, err)
	}
}

func TestSnapshotCopiesConditionsAndOrdering(t *testing.T) {
	r := reg("a")
	r.Definition.DependsOn = []string{"b", "c"}
	r.Conditions = []bean.Condition{
		{
			Missing: gs.Selector{
				Contract:   "int",
				Qualifiers: []string{"missing"},
			},
			Present: gs.Selector{
				Contract:   "int",
				Qualifiers: []string{"present"},
			},
		},
	}
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:            "batch",
		Schema:        bean.SchemaVersion,
		Registrations: []bean.Registration{r},
	})
	r.Definition.DependsOn[0] = "changed"
	r.Conditions[0].Missing.Qualifiers[0] = "changed"
	r.Conditions[0].Present.Qualifiers[0] = "changed"
	_, rs, err := c.Snapshot("")
	if err != nil {
		t.Fatal(err)
	}
	got := rs[0]
	if got.Definition.DependsOn[0] != "b" ||
		got.Definition.DependsOn[1] != "c" ||
		got.Conditions[0].Missing.Qualifiers[0] != "missing" ||
		got.Conditions[0].Present.Qualifiers[0] != "present" {
		t.Fatal(got)
	}
}

func TestConditionSelectorCannotContainQualifiersAlone(t *testing.T) {
	for _, missing := range []bool{true, false} {
		r := reg("a")
		condition := bean.Condition{}
		if missing {
			condition.Missing.Qualifiers = []string{"q"}
		} else {
			condition.Present.Qualifiers = []string{"q"}
		}
		r.Conditions = []bean.Condition{condition}
		c := catalog.New()
		c.Contribute(bean.Batch{
			ID:            "b",
			Schema:        bean.SchemaVersion,
			Registrations: []bean.Registration{r},
		})
		if _, _, err := c.Snapshot(""); err == nil {
			t.Fatal("unconstrained condition selector accepted")
		}
	}
}

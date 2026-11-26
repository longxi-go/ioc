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
	"testing"

	"github.com/longxi-go/ioc/pkg/bean"
	managed "github.com/longxi-go/ioc/pkg/context"
)

func TestApplicationConditionReportUsesPropertySnapshot(t *testing.T) {
	r := intRecipe("singleton")
	r.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"enabled"},
				HavingValue: new("yes"),
			},
		},
	}
	properties := map[string]string{"enabled": "yes"}
	a := testApplication(t, r, managed.WithProperties(properties))
	properties["enabled"] = "no"
	report, err := a.ConditionReport()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, candidate := range report.Candidates {
		if candidate.Bean == "integer" {
			found = true
			if candidate.Outcome != bean.Accepted ||
				len(candidate.Predicates) != 1 ||
				!candidate.Predicates[0].Matched {
				t.Fatal(candidate)
			}
		}
	}
	if !found {
		t.Fatal("missing candidate")
	}
	if err := a.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ConditionReport(); err != nil {
		t.Fatal(err)
	}
}

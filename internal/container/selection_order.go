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
	"cmp"
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	cyclicBeanConditionObservationsMessage = "cyclic bean condition observations"
)

func ordered(rs []bean.Registration) {
	slices.SortFunc(rs, func(a, b bean.Registration) int {
		return cmp.Compare(a.Definition.ID, b.Definition.ID)
	})
}

func selectorSet(s gs.Selector) bool {
	return s.Name != "" || s.Contract != "" || len(s.Qualifiers) > 0
}

// selectionOrder evaluates observations after their possible contributors.
// Property-ineligible candidates must already be removed by the selection pass.
func selectionOrder(rs []bean.Registration) ([]int, error) {
	edges := make([]map[int]bool, len(rs))
	for i := range edges {
		edges[i] = map[int]bool{}
	}
	for i, r := range rs {
		for _, c := range r.Conditions {
			for _, s := range []gs.Selector{c.Missing, c.Present} {
				if !selectorSet(s) {
					continue
				}
				for j, other := range rs {
					if i != j && other.Definition.Matches(s) {
						edges[j][i] = true
					}
				}
			}
		}
	}
	order, err := topological(edges)
	if err != nil {
		diagnostic := graphFailure(rs, edges)
		diagnostic.Message = cyclicBeanConditionObservationsMessage
		return nil, diagnostic
	}
	return order, nil
}

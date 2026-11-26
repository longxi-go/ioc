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
	"slices"

	"github.com/longxi-go/gosr/pkg/environment"

	"github.com/longxi-go/ioc/pkg/bean"
)

func selectRecipes(rs []bean.Registration, snapshot environment.Snapshot) ([]bean.Registration, error) {
	accepted, _, err := evaluateSelection(rs, snapshot)
	return accepted, err
}

// evaluateSelection is shared by binding and diagnostics. All observations use
// copied metadata; neither path may construct objects or invoke user callbacks.
func evaluateSelection(rs []bean.Registration, snapshot environment.Snapshot) ([]bean.Registration, bean.ConditionReport, error) {
	rs = slices.Clone(rs)
	ordered(rs)
	report := bean.ConditionReport{
		Candidates: make([]bean.CandidateEvaluation, len(rs)),
	}
	positions := make(map[string]int, len(rs))
	eligible := make([]bean.Registration, 0, len(rs))
	for i, r := range rs {
		evaluation, err := evaluateProperties(r, snapshot)
		report.Candidates[i] = evaluation
		if err != nil {
			return nil, report, err
		}
		positions[r.Definition.ID] = i
		// Property exclusions precede ordering, so mutually exclusive
		// configurations cannot introduce fictitious observation cycles.
		if report.Candidates[i].Outcome != bean.Rejected {
			eligible = append(eligible, r)
		}
	}
	order, err := selectionOrder(eligible)
	if err != nil {
		return nil, report, err
	}
	var accepted []bean.Registration
	for _, i := range order {
		r := eligible[i]
		result := &report.Candidates[positions[r.Definition.ID]]
		evaluateBeanConditions(r, accepted, result)
		if result.Outcome == bean.Accepted {
			accepted = append(accepted, r)
		}
	}
	return accepted, report, nil
}

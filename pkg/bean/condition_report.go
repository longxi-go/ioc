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

package bean

import gs "github.com/longxi-go/gosr/pkg/ioc"

// ConditionOutcome distinguishes rejection from an ordering failure that
// prevented the evaluator from reaching a candidate.
type ConditionOutcome string

const (
	Accepted     ConditionOutcome = "accepted"
	Rejected     ConditionOutcome = "rejected"
	NotEvaluated ConditionOutcome = "not-evaluated"
)

// PredicateKind identifies the operation without exposing executable recipes.
type PredicateKind string

const (
	PropertyPredicate      PredicateKind = "property"
	ProfilePredicate       PredicateKind = "profile"
	OnBeanPredicate        PredicateKind = "on-bean"
	OnMissingBeanPredicate PredicateKind = "on-missing-bean"
)

// PredicateEvaluation records an observed predicate. Property comparisons expose
// only their declared address and result; interpolated addresses and compared
// values may contain credentials and are never printed.
// Matches contains accepted Bean IDs in observation order, before this candidate.
type PredicateEvaluation struct {
	Kind     PredicateKind
	Property string
	Selector gs.Selector
	Matched  bool
	Matches  []string
}

// CandidateEvaluation describes selection, not construction or activation.
// Property-rejected candidates have only property observations; candidates left
// NotEvaluated have not observed Bean predicates because ordering failed.
type CandidateEvaluation struct {
	Bean       string
	Source     gs.Source
	Outcome    ConditionOutcome
	Predicates []PredicateEvaluation
}

// ConditionReport owns detached metadata from one deterministic selection pass.
// An accepted candidate can still fail dependency binding, construction or Start.
// Candidates follow ID for presentation; Matches follows observation order.
type ConditionReport struct {
	Candidates []CandidateEvaluation
}

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
	"errors"
	"slices"
	"strings"

	"github.com/longxi-go/gosr/pkg/environment"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	disabledPropertyValue = "false"

	conditionEnvironmentEvaluationFailedMessage   = "condition environment evaluation failed"
	conditionEnvironmentOperationFailedMessage    = "condition environment operation failed"
	conditionRequiresAScalarPropertyMessage       = "condition requires a scalar property"
	conditionRequiresAnEnvironmentSnapshotMessage = "condition requires an environment snapshot"
)

type conditionCause struct {
	cause error
}

func (e conditionCause) Error() string {
	return conditionEnvironmentOperationFailedMessage
}

func (e conditionCause) Unwrap() error {
	return e.cause
}

func evaluateProperties(r bean.Registration, snapshot environment.Snapshot) (bean.CandidateEvaluation, error) {
	result := bean.CandidateEvaluation{
		Bean:    r.Definition.ID,
		Source:  r.Definition.Source,
		Outcome: bean.NotEvaluated,
	}
	fail := func(err error) (bean.CandidateEvaluation, error) {
		return result, &gs.Error{
			Code:    gs.InvalidDefinition,
			Bean:    r.Definition.ID,
			Source:  r.Definition.Source,
			Message: conditionEnvironmentEvaluationFailedMessage,
			Cause: conditionCause{
				err,
			},
		}
	}
	observe := func(kind bean.PredicateKind, key string, matched bool) {
		result.Predicates = append(result.Predicates, bean.PredicateEvaluation{
			Kind:     kind,
			Property: key,
			Matched:  matched,
		})
		if !matched {
			result.Outcome = bean.Rejected
		}
	}
	for _, c := range r.Conditions {
		if snapshot == nil && (c.Property != nil || len(c.Profiles) > 0) {
			return fail(errors.New(conditionRequiresAnEnvironmentSnapshotMessage))
		}
		if p := c.Property; p != nil {
			expected := ""
			if p.HavingValue != nil {
				var err error
				expected, err = snapshot.Resolve(*p.HavingValue)
				if err != nil {
					return fail(err)
				}
			}
			prefix, err := snapshot.Resolve(p.Prefix)
			if err != nil {
				return fail(err)
			}
			prefix = strings.TrimSuffix(prefix, bean.PropertyConditionSeparator)
			for _, name := range p.Names {
				declared := name
				if p.Prefix != "" {
					declared = strings.TrimSuffix(p.Prefix, bean.PropertyConditionSeparator) + bean.PropertyConditionSeparator + name
				}
				resolvedName, err := snapshot.Resolve(name)
				if err != nil {
					return fail(err)
				}
				key := resolvedName
				if prefix != "" {
					key = prefix + bean.PropertyConditionSeparator + resolvedName
				}
				if err := environment.ValidateKey(key); err != nil {
					return fail(err)
				}

				property, present, err := snapshot.Lookup(key)
				if err != nil {
					return fail(err)
				}
				matched := p.MatchIfMissing
				if present {
					scalar, ok := property.Value.Scalar()
					if !ok {
						return fail(errors.New(conditionRequiresAScalarPropertyMessage))
					}
					matched = !strings.EqualFold(scalar, disabledPropertyValue)
					if p.HavingValue != nil {
						matched = scalar == expected
					}
				}
				observe(bean.PropertyPredicate, declared, matched)
			}
		}
		if len(c.Profiles) > 0 {
			matched, err := snapshot.AcceptsProfiles(c.Profiles...)
			if err != nil {
				return fail(err)
			}
			observe(bean.ProfilePredicate, "", matched)
		}
	}
	return result, nil
}

func evaluateBeanConditions(r bean.Registration, accepted []bean.Registration, result *bean.CandidateEvaluation) {
	result.Outcome = bean.Accepted
	for _, c := range r.Conditions {
		for _, kind := range []bean.PredicateKind{bean.OnMissingBeanPredicate, bean.OnBeanPredicate} {
			selector := c.Missing
			if kind == bean.OnBeanPredicate {
				selector = c.Present
			}
			if !selectorSet(selector) {
				continue
			}
			predicate := observeSelector(kind, selector, accepted)
			result.Predicates = append(result.Predicates, predicate)
			if !predicate.Matched {
				result.Outcome = bean.Rejected
			}
		}
	}
}

func observeSelector(kind bean.PredicateKind, selector gs.Selector, accepted []bean.Registration) bean.PredicateEvaluation {
	selector.Qualifiers = slices.Clone(selector.Qualifiers)
	result := bean.PredicateEvaluation{
		Kind:     kind,
		Selector: selector,
	}
	for _, candidate := range accepted {
		if candidate.Definition.Matches(selector) {
			result.Matches = append(result.Matches, candidate.Definition.ID)
		}
	}
	result.Matched = len(result.Matches) > 0
	if kind == bean.OnMissingBeanPredicate {
		result.Matched = !result.Matched
	}
	return result
}

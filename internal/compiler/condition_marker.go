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

package compiler

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"strings"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"golang.org/x/tools/go/packages"
)

const (
	conditionErrorFormat = "%s: %w"

	conditionNameMustBeNonblankMessage = "condition name must be nonblank"
	conditionTypeLabel                 = "condition type"
	profileRequiresPredicatesMessage   = "profile requires predicates"
)

// Pointers distinguish omitted attributes (infer the result type) from explicit
// blanks (diagnose a configuration mistake rather than changing its meaning).
type conditionalMarker struct {
	Type *string `marker:",optional"`
	Name *string `marker:",optional"`
}

func compileConditions(pkg *packages.Package, pos token.Pos, result types.Type, a annotation) ([]condition, error) {
	var conditions []condition
	for _, value := range a[markerOnProperty] {
		m := value.(propertyMarker)
		c := condition{
			Property: &bean.PropertyCondition{
				Prefix:         m.Prefix,
				Names:          m.Name,
				HavingValue:    m.HavingValue,
				MatchIfMissing: m.MatchIfMissing,
			},
		}
		if err := c.Validate(); err != nil {
			return nil, err
		}
		conditions = append(conditions, c)
	}
	if a.flag(markerProfile) {
		c := condition{
			Profiles: a[markerProfile][0].([]string),
		}
		if len(c.Profiles) == 0 {
			return nil, errors.New(profileRequiresPredicatesMessage)
		}
		if err := c.Validate(); err != nil {
			return nil, err
		}
		conditions = append(conditions, c)
	}
	for _, key := range []string{markerOnBean, markerOnMissingBean} {
		for _, value := range a[key] {
			marker := value.(conditionalMarker)
			selectors, err := marker.selectors(pkg, pos, result)
			if err != nil {
				return nil, fmt.Errorf(conditionErrorFormat, annotationPrefix+key, err)
			}
			// Each attribute observes the accepted definitions independently;
			// combining them into one Selector would require the same Bean.
			for _, selector := range selectors {
				c := condition{}
				if key == markerOnBean {
					c.Present = selector
				} else {
					c.Missing = selector
				}
				conditions = append(conditions, c)
			}
		}
	}
	return conditions, nil
}

func (m conditionalMarker) selectors(pkg *packages.Package, pos token.Pos, result types.Type) ([]gs.Selector, error) {
	if m.Type == nil && m.Name == nil {
		return []gs.Selector{
			{
				Contract: contract(result),
			},
		}, nil
	}
	var selectors []gs.Selector
	if m.Type != nil {
		t, err := resolveTypeReference(pkg, pos, *m.Type, conditionTypeLabel)
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, gs.Selector{
			Contract: contract(t),
		})
	}
	if m.Name != nil {
		if strings.TrimSpace(*m.Name) == "" {
			return nil, errors.New(conditionNameMustBeNonblankMessage)
		}
		selectors = append(selectors, gs.Selector{
			Name: *m.Name,
		})
	}
	return selectors, nil
}

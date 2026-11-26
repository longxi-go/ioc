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

import (
	"errors"
	"strings"

	"github.com/longxi-go/gosr/pkg/environment"
	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	conditionRequiresAPropertyOrBeanSelectorMessage = "condition requires a property or bean selector"
	invalidPropertyNamesMessage                     = "property condition requires nonblank unique names"
	invalidPropertyPrefixMessage                    = "property condition requires a nonblank prefix with at most one trailing dot"
	propertyConditionRequiresNamesMessage           = "property condition requires names"
)

// PropertyConditionSeparator defines prefix/name composition for IoC conditions.
// Gosr property keys remain exact addresses; this does not add key normalization.
const PropertyConditionSeparator = "."

// PropertyCondition groups exact property addresses under an optional prefix.
// HavingValue preserves the difference between presence and an empty comparison.
type PropertyCondition struct {
	Prefix         string
	Names          []string
	HavingValue    *string
	MatchIfMissing bool
}

// Condition combines property, profile and accepted-definition predicates with AND.
// Metadata stays declarative so selection never constructs candidates.
type Condition struct {
	Property         *PropertyCondition
	Profiles         []string
	Missing, Present gs.Selector
}

// Validate rejects metadata that could silently become unconditional.
func (c Condition) Validate() error {
	populated := c.Property != nil || len(c.Profiles) > 0
	if p := c.Property; p != nil {
		if len(p.Names) == 0 {
			return errors.New(propertyConditionRequiresNamesMessage)
		}
		prefix := strings.TrimSuffix(p.Prefix, PropertyConditionSeparator)
		if p.Prefix != "" &&
			(prefix == "" || strings.TrimSpace(prefix) != prefix || strings.HasSuffix(prefix, PropertyConditionSeparator)) {
			return errors.New(invalidPropertyPrefixMessage)
		}
		seen := map[string]bool{}
		for _, name := range p.Names {
			if strings.TrimSpace(name) == "" || seen[name] {
				return errors.New(invalidPropertyNamesMessage)
			}
			seen[name] = true
			key := name
			if prefix != "" {
				key = prefix + PropertyConditionSeparator + name
			}
			if !strings.Contains(key, environment.PlaceholderPrefix) {
				if err := environment.ValidateKey(key); err != nil {
					return err
				}
			}
		}
	}
	for _, profile := range c.Profiles {
		if _, _, err := environment.ParseProfilePredicate(profile); err != nil {
			return err
		}
	}
	for _, selector := range []gs.Selector{c.Missing, c.Present} {
		if selector.Contract == "" && selector.Name == "" && len(selector.Qualifiers) == 0 {
			continue
		}
		populated = true
		if err := selector.Validate(); err != nil {
			return err
		}
	}
	if !populated {
		return errors.New(conditionRequiresAPropertyOrBeanSelectorMessage)
	}
	return nil
}

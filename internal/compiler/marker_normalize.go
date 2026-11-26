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
	"strings"

	env "github.com/longxi-go/gosr/pkg/environment"
	"github.com/longxi-go/ioc/internal/httpcompiler"
)

const (
	staticAnnotationValueCannotContainPlaceholdersFormat = "static annotation value %q cannot contain placeholders"
	staticAnnotationValueMustBeNonblankMessage           = "static annotation value must be nonblank"
)

// normalizeMarkers translates structured source attributes to the internal selector
// representation only after parsing, so legacy supplements remain unknown syntax.
func normalizeMarkers(a annotation) error {
	for _, key := range []string{markerBean, markerComponent, httpcompiler.Controller} {
		if !a.flag(key) {
			continue
		}
		m := a[key][0].(registrationMarker)
		if m.Name != nil {
			if err := validateStaticMarkerValue(*m.Name); err != nil {
				return err
			}
			a[markerName] = []any{*m.Name}
		}
		for _, alias := range m.Aliases {
			if err := validateStaticMarkerValue(alias); err != nil {
				return err
			}
			a[markerAlias] = append(a[markerAlias], alias)
		}
	}
	if a.flag(markerInject) {
		m := a[markerInject][0].(injectionMarker)
		if m.Name != nil {
			if err := validateStaticMarkerValue(*m.Name); err != nil {
				return err
			}
			a[markerName] = []any{*m.Name}
		}
		for _, q := range m.Qualifiers {
			if err := validateStaticMarkerValue(q); err != nil {
				return err
			}
			a[markerQualifier] = append(a[markerQualifier], q)
		}
	}
	for _, key := range []string{markerName, markerAlias, markerQualifier, markerScope, markerExpose, markerDependsOn} {
		for _, value := range a[key] {
			if err := validateStaticMarkerValue(value.(string)); err != nil {
				return err
			}
		}
	}
	for _, key := range []string{markerOnBean, markerOnMissingBean} {
		for _, value := range a[key] {
			m := value.(conditionalMarker)
			for _, v := range []*string{m.Name, m.Type} {
				if v != nil {
					if err := validateStaticMarkerValue(*v); err != nil {
						return err
					}
				}
			}
		}
	}
	for _, value := range a[markerParam] {
		m := value.(parameterMarker)
		for _, v := range append([]string{m.Parameter}, m.Qualifiers...) {
			if err := validateStaticMarkerValue(v); err != nil {
				return err
			}
		}
		if m.Name != "" {
			if err := validateStaticMarkerValue(m.Name); err != nil {
				return err
			}
		}
	}
	if a.flag(markerProfile) {
		for _, p := range a[markerProfile][0].([]string) {
			if err := validateStaticMarkerValue(p); err != nil {
				return err
			}
			if _, _, err := env.ParseProfilePredicate(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateStaticMarkerValue(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(staticAnnotationValueMustBeNonblankMessage)
	}
	if strings.Contains(value, env.PlaceholderPrefix) {
		return fmt.Errorf(staticAnnotationValueCannotContainPlaceholdersFormat, value)
	}
	return nil
}

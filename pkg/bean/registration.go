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
	"slices"
	"strings"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	applicationListenersRequireSingletonOwnershipMessage = "application listeners require singleton ownership"
	duplicateListenerDeclarationMessage                  = "duplicate listener declaration "
	invalidConditionPredicateMessage                     = "invalid condition predicate"
	invalidListenerMessage                               = "listener requires identity, event type, valid source and static adapters"

	reservedEnvironmentIdentityCannotBeContributedMessage    = "reserved Environment identity cannot be contributed"
	reservedEventPublisherIdentityCannotBeContributedMessage = "reserved event publisher identity cannot be contributed"
)

// Validate checks contributed recipe structure without executing user code.
// Active graph, lifecycle matching and conditional primary selection remain
// runtime/compiler responsibilities; infrastructure is installed by the owner.
func (r Registration) Validate() error {
	if err := r.Definition.Validate(); err != nil {
		return err
	}
	invalid := func(message string) error {
		return &gs.Error{
			Code:    gs.InvalidDefinition,
			Bean:    r.Definition.ID,
			Source:  r.Definition.Source,
			Message: message,
		}
	}
	if r.Create == nil {
		return invalid(MissingFactoryMessage)
	}
	d := r.Definition
	if d.ID == EventPublisherID ||
		d.Name == EventPublisherName ||
		slices.Contains(d.Aliases, EventPublisherName) {
		return invalid(reservedEventPublisherIdentityCannotBeContributedMessage)
	}
	if d.ID == EnvironmentID || d.Name == EnvironmentName || slices.Contains(d.Aliases, EnvironmentName) {
		return invalid(reservedEnvironmentIdentityCannotBeContributedMessage)
	}
	if len(r.Listeners) > 0 && d.Scope != gs.Singleton {
		return invalid(applicationListenersRequireSingletonOwnershipMessage)
	}
	seen := map[string]bool{}
	for _, listener := range r.Listeners {
		if strings.TrimSpace(listener.ID) == "" ||
			strings.TrimSpace(listener.Event) == "" ||
			listener.Source.Line < 0 ||
			listener.Match == nil ||
			listener.Invoke == nil {
			return invalid(invalidListenerMessage)
		}
		if seen[listener.ID] {
			return invalid(duplicateListenerDeclarationMessage + listener.ID)
		}
		seen[listener.ID] = true
	}
	for _, condition := range r.Conditions {
		if err := condition.Validate(); err != nil {
			return &gs.Error{
				Code:    gs.InvalidDefinition,
				Bean:    d.ID,
				Source:  d.Source,
				Message: invalidConditionPredicateMessage,
				Cause:   err,
			}
		}
	}
	return nil
}

// Clone transfers ownership of all mutable recipe metadata. Function values
// remain static adapters; their captured state is not cloned or made immutable.
func (r Registration) Clone() Registration {
	r.Definition = r.Definition.Clone()
	r.Conditions = slices.Clone(r.Conditions)
	for i := range r.Conditions {
		r.Conditions[i].Profiles = slices.Clone(r.Conditions[i].Profiles)
		if p := r.Conditions[i].Property; p != nil {
			cloned := *p
			cloned.Names = slices.Clone(p.Names)
			if p.HavingValue != nil {
				value := *p.HavingValue
				cloned.HavingValue = &value
			}
			r.Conditions[i].Property = &cloned
		}
		r.Conditions[i].Missing.Qualifiers = slices.Clone(r.Conditions[i].Missing.Qualifiers)
		r.Conditions[i].Present.Qualifiers = slices.Clone(r.Conditions[i].Present.Qualifiers)
	}
	r.Listeners = slices.Clone(r.Listeners)
	for i := range r.Listeners {
		if r.Listeners[i].Order != nil {
			value := *r.Listeners[i].Order
			r.Listeners[i].Order = &value
		}
	}
	return r
}

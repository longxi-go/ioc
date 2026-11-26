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
	"strings"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	invalidPublishedListenerOwnershipMessage              = "published listener has invalid singleton/lifecycle ownership"
	conditionalFlagDisagreesWithConditionsMessage         = "conditional flag disagrees with conditions"
	contractTypeTableIncompleteMessage                    = "contract type table incomplete"
	definitionMissingAdapterPackageMessage                = "definition missing adapter package"
	duplicateDefinitionMessage                            = "duplicate definition"
	invalidOrDuplicatePublishedListenerDeclarationMessage = "invalid or duplicate published listener declaration"
	invalidPublishedConditionMessage                      = "invalid published condition: "
)

func (v *publicationValidation) recipeRecords() error {
	manifest, packages, definitions := v.manifest, v.packages, v.definitions
	for _, r := range manifest.Recipes {
		if r.Conditional != (len(r.Conditions) > 0) {
			return errors.New(conditionalFlagDisagreesWithConditionsMessage)
		}
		for _, condition := range r.Conditions {
			if err := condition.Validate(); err != nil {
				return errors.New(invalidPublishedConditionMessage + err.Error())
			}
		}
		if !packages[r.Definition.Source.Package] {
			return errors.New(definitionMissingAdapterPackageMessage)
		}
		if definitions[r.Definition.ID] {
			return errors.New(duplicateDefinitionMessage)
		}
		definitions[r.Definition.ID] = true
		if len(r.Types) != len(r.Definition.Contracts) {
			return errors.New(contractTypeTableIncompleteMessage)
		}
		if err := r.Definition.Validate(); err != nil {
			return errors.New(err.Error())
		}
		seenListeners := map[string]bool{}
		for _, listener := range r.Listeners {
			if !strings.HasPrefix(listener.ID, r.Definition.ID+definitionMemberSeparator) ||
				strings.TrimSpace(listener.Event) == "" ||
				listener.Source.Line < 0 ||
				seenListeners[listener.ID] {
				return errors.New(invalidOrDuplicatePublishedListenerDeclarationMessage)
			}
			seenListeners[listener.ID] = true
			if r.Definition.Scope != gs.Singleton || listener.Lifecycle && r.Definition.Lazy {
				return errors.New(invalidPublishedListenerOwnershipMessage)
			}
		}
	}

	return nil
}

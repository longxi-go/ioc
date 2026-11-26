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
	"fmt"
)

const (
	duplicateDefinitionFormat = "duplicate definition %s"
	duplicateActiveNameFormat = "duplicate active name %s (%s, %s)"
)

func (g *graphValidation) index() error {
	names := map[string]string{}
	for _, r := range g.recipes {
		if _, exists := g.byID[r.Definition.ID]; exists {
			return fmt.Errorf(duplicateDefinitionFormat, r.Definition.ID)
		}
		g.byID[r.Definition.ID] = r
		if !r.Conditional {
			for _, name := range append([]string{r.Definition.Name}, r.Definition.Aliases...) {
				if old, exists := names[name]; exists {
					return fmt.Errorf(duplicateActiveNameFormat, name, old, r.Definition.ID)
				}
				names[name] = r.Definition.ID
			}
		}
	}
	return nil
}

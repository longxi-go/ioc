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
)

const (
	incompletePublicationClosureMessage = "annotation/index closure is incomplete"
	unindexedAnnotationMessage          = "unindexed annotation: "
	unrecordedIndexImportMessage        = "unrecorded index import: "
)

func (v *publicationValidation) closure() error {
	imports, packages, definitions, mod := v.imports, v.packages, v.definitions, v.module
	env, tags := v.target, v.tags
	for imported := range imports {
		if !packages[imported] {
			return errors.New(unrecordedIndexImportMessage + imported)
		}
	}
	declared, err := publicationDeclarations(mod.Dir, mod.Path, env, tags)
	if err != nil {
		return errors.New(err.Error())
	}
	if len(declared) != len(definitions) {
		return errors.New(incompletePublicationClosureMessage)
	}
	for id := range declared {
		if !definitions[id] {
			return errors.New(unindexedAnnotationMessage + id)
		}
	}

	return nil
}

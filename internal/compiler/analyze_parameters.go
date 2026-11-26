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
)

const (
	blankIdentifier                                = "_"
	contextParameterCannotHaveAnIoCSelectorMessage = "context parameter cannot have an IoC selector"
	duplicateParamBindingFormat                    = "duplicate param binding %s"
	paramRequiresAnExistingNamedParameterMessage   = "param requires an existing named parameter"
	paramSelectorNamesAnUnknownParameterMessage    = "param selector names an unknown parameter"
	paramSelectorRequiresANameOrQualifiersMessage  = "param selector requires a name or qualifiers"
)

type parameterAnalysis struct {
	signature *types.Signature
	markers   annotation
	position  token.Pos
}

func (analysis *packageAnalysis) parameters(r *recipe, input parameterAnalysis) error {
	pkg, sig, a, pos := analysis.pkg, input.signature, input.markers, input.position
	bindings := map[string]parameterMarker{}
	for _, value := range a[markerParam] {
		binding := value.(parameterMarker)
		if binding.Parameter == "" || binding.Parameter == blankIdentifier {
			return at(pkg, pos, errors.New(paramRequiresAnExistingNamedParameterMessage))
		}
		if _, exists := bindings[binding.Parameter]; exists {
			return at(pkg, pos, fmt.Errorf(duplicateParamBindingFormat, binding.Parameter))
		}
		bindings[binding.Parameter] = binding
	}
	for i := 0; i < sig.Params().Len(); i++ {
		param := sig.Params().At(i)
		selectors := annotation{}
		if binding, exists := bindings[param.Name()]; exists {
			if isContext(param.Type()) {
				return at(pkg, param.Pos(), errors.New(contextParameterCannotHaveAnIoCSelectorMessage))
			}
			if binding.Name != "" {
				selectors[markerName] = []any{binding.Name}
			}
			for _, q := range binding.Qualifiers {
				selectors[markerQualifier] = append(selectors[markerQualifier], q)
			}
			if binding.Name == "" && len(binding.Qualifiers) == 0 {
				return at(pkg, param.Pos(), errors.New(paramSelectorRequiresANameOrQualifiersMessage))
			}
			delete(bindings, param.Name())
		}
		if err := r.addArg(param.Type(), "", selectors, source(pkg, param.Pos())); err != nil {
			return at(pkg, param.Pos(), err)
		}
	}
	if len(bindings) > 0 {
		return at(pkg, pos, errors.New(paramSelectorNamesAnUnknownParameterMessage))
	}
	return nil
}

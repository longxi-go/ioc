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
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	mustBeAGoTypeFormat                  = "%s %q must be a Go type"
	mustBeAValueTypeNotAConstraintFormat = "%s %q must be a value type, not a constraint"
	mustBeNonblankFormat                 = "%s must be nonblank"
	requiresGenericTypeArgumentsFormat   = "%s %q requires generic type arguments"
	resolveFormat                        = "resolve %s %q: %w"
)

// resolveTypeReference uses the declaration's file scope, including import
// aliases. Package-name guesses and value expressions cannot become contracts.
func resolveTypeReference(pkg *packages.Package, pos token.Pos, text, purpose string) (types.Type, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf(mustBeNonblankFormat, purpose)
	}
	value, err := types.Eval(pkg.Fset, pkg.Types, pos, text)
	if err != nil {
		return nil, fmt.Errorf(resolveFormat, purpose, text, err)
	}
	if !value.IsType() {
		return nil, fmt.Errorf(mustBeAGoTypeFormat, purpose, text)
	}
	// Inspect the original alias before unaliasing: even an unused generic
	// parameter (type Phantom[T any] = int) still requires instantiation.
	if requiresTypeArguments(value.Type) {
		return nil, fmt.Errorf(requiresGenericTypeArgumentsFormat, purpose, text)
	}
	t := types.Unalias(value.Type)
	if iface, ok := t.Underlying().(*types.Interface); ok && !iface.IsMethodSet() {
		return nil, fmt.Errorf(mustBeAValueTypeNotAConstraintFormat, purpose, text)
	}
	return t, nil
}

func requiresTypeArguments(t types.Type) bool {
	switch ref := t.(type) {
	case *types.Alias:
		return ref.TypeParams().Len() > 0 && ref.TypeArgs().Len() == 0
	case *types.Named:
		return ref.TypeParams().Len() > 0 && ref.TypeArgs().Len() == 0
	default:
		return false
	}
}

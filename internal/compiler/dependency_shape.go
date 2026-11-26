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
	"go/types"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	unsupportedDependencyShapeFormat = "unsupported dependency shape %s: use T, Optional[T], []T, Provider[T], Provider[Optional[T]] or Provider[[]T]; provider must be the outermost wrapper"
)

// dependencyKind describes injection syntax independently of a Go type's
// underlying representation.
type dependencyKind string

const (
	valueShape    dependencyKind = "value"
	providerShape dependencyKind = "provider"
	optionalShape dependencyKind = "optional"
	manyShape     dependencyKind = "many"
)

// dependencyShape interprets only the outer injection operators. Named
// collections and the contents of exact map/array/pointer contracts remain opaque.
func dependencyShape(t types.Type) (*shape, gs.Access, gs.Cardinality, error) {
	root := injectionOperator(t)
	current := root
	access, cardinality := gs.Direct, gs.One
	if current.Kind == providerShape {
		access = gs.Deferred
		current.Elem = injectionOperator(operatorElement(current.Type))
		current = current.Elem
	}
	switch current.Kind {
	case optionalShape:
		cardinality = gs.ZeroOrOne
	case manyShape:
		cardinality = gs.Many
	case providerShape:
		return nil, "", "", invalidDependencyShape(t)
	}
	if current.Kind != valueShape {
		current.Elem = injectionOperator(operatorElement(current.Type))
		if current.Elem.Kind != valueShape {
			return nil, "", "", invalidDependencyShape(t)
		}
	}
	return root, access, cardinality, nil
}

func invalidDependencyShape(t types.Type) error {
	return fmt.Errorf(unsupportedDependencyShapeFormat, t)
}

func injectionOperator(t types.Type) *shape {
	t = types.Unalias(t)
	s := &shape{
		Kind: valueShape,
		Type: t,
	}
	if n, ok := t.(*types.Named); ok && n.Obj().Pkg() != nil &&
		n.Obj().Pkg().Path() == contractsPackage && n.TypeArgs().Len() == 1 {
		switch n.Obj().Name() {
		case providerType:
			s.Kind = providerShape
		case optionalType:
			s.Kind = optionalShape
		}
	}
	if _, ok := t.(*types.Slice); ok {
		s.Kind = manyShape
	}
	return s
}

func operatorElement(t types.Type) types.Type {
	if slice, ok := t.(*types.Slice); ok {
		return slice.Elem()
	}
	return t.(*types.Named).TypeArgs().At(0)
}

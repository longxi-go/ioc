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
	"go/types"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

func (r *recipe) addArg(t types.Type, field string, a annotation, src gs.Source) error {
	arg := argument{
		Field:   field,
		Context: isContext(t),
		Port:    len(r.Definition.Dependencies),
	}
	if !arg.Context {
		var access gs.Access
		var card gs.Cardinality
		var err error
		arg.Shape, access, card, err = dependencyShape(t)
		if err != nil {
			return err
		}
		r.Definition.Dependencies = append(r.Definition.Dependencies, gs.Dependency{
			Selector: gs.Selector{
				Contract:   contract(leaf(arg.Shape)),
				Name:       a.one(markerName),
				Qualifiers: a.strings(markerQualifier),
			},
			Access:      access,
			Cardinality: card,
			Source:      src,
		})
	}
	r.Args = append(r.Args, arg)
	return nil
}

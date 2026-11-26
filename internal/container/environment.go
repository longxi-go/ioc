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

package container

import (
	"context"
	"slices"

	env "github.com/longxi-go/gosr/pkg/environment"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	reservedEnvironmentPrimaryMessage = "the application Environment is already primary; select alternatives by name"
)

func (e *Engine) environmentRecipe() bean.Registration {
	return bean.Registration{
		Definition: gs.BeanDefinition{
			ID:        bean.EnvironmentID,
			Name:      bean.EnvironmentName,
			Scope:     gs.Singleton,
			Primary:   true,
			Contracts: []gs.Contract{bean.EnvironmentContract},
		},
		Create: e.environmentFactory,
		Accept: acceptEnvironment,
	}
}

func (e *Engine) environmentFactory(context.Context, bean.Ports) (any, error) {
	return e.opts.Environment, nil
}

func acceptEnvironment(value any) bool {
	_, ok := value.(env.Environment)
	return ok
}

func validateEnvironmentPrimary(bs []*bound) error {
	for _, b := range bs {
		r := b.recipe
		if r.Definition.ID != bean.EnvironmentID &&
			r.Definition.Primary &&
			slices.Contains(r.Definition.Contracts, bean.EnvironmentContract) {
			return recipeFailure(r, gs.InvalidDefinition, reservedEnvironmentPrimaryMessage, nil)
		}
	}
	return nil
}

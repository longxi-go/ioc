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
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	reservedInfrastructureIdentityFormat = "%s: reserved infrastructure identity %s"
	reservedInfrastructurePrimaryFormat  = "%s: infrastructure is already primary; select alternatives by name"
)

const environmentPackage = contractsModule + "/pkg/environment"

// infrastructureGraph validates reserved identities and adds context-owned
// capabilities to static binding and query analysis, never library contributions.
func infrastructureGraph(recipes []publishedRecipe) ([]publishedRecipe, error) {
	infrastructure := []publishedRecipe{
		infrastructureRecipe(infrastructureSpec{
			ID:       bean.EventPublisherID,
			Name:     bean.EventPublisherName,
			Contract: bean.EventPublisherContract,
			Package:  eventPackage,
			Type:     publisherType,
		}),
		infrastructureRecipe(infrastructureSpec{
			ID:       bean.EnvironmentID,
			Name:     bean.EnvironmentName,
			Contract: bean.EnvironmentContract,
			Package:  environmentPackage,
			Type:     environmentType,
		}),
	}
	for _, r := range recipes {
		d := r.Definition
		for _, builtin := range infrastructure {
			reserved := builtin.Definition
			if d.ID == reserved.ID || d.Name == reserved.Name || slices.Contains(d.Aliases, reserved.Name) {
				return nil, fmt.Errorf(reservedInfrastructureIdentityFormat, d.ID, reserved.Name)
			}
			if !r.Conditional && d.Primary && slices.Contains(d.Contracts, reserved.Contracts[0]) {
				return nil, fmt.Errorf(reservedInfrastructurePrimaryFormat, d.ID)
			}
		}
	}
	return append(slices.Clone(recipes), infrastructure...), nil
}

type infrastructureSpec struct {
	ID       string
	Name     string
	Contract gs.Contract
	Package  string
	Type     string
}

func infrastructureRecipe(spec infrastructureSpec) publishedRecipe {
	return publishedRecipe{
		Definition: gs.BeanDefinition{
			ID:        spec.ID,
			Name:      spec.Name,
			Contracts: []gs.Contract{spec.Contract},
			Scope:     gs.Singleton,
			Primary:   true,
		},
		Types: []typeSpec{
			{
				Code:    typeAlias(spec.Package) + goSelector + spec.Type,
				Imports: map[string]string{spec.Package: typeAlias(spec.Package)},
			},
		},
	}
}

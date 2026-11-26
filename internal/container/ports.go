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

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	invalidDependencyPortMessage = "invalid dependency port"
)

type ports struct {
	engine *Engine
	bound  *bound
}

func (p ports) Resolve(ctx context.Context, index int) ([]any, error) {
	if index < 0 || index >= len(p.bound.ports) {
		return nil, failure(gs.InvalidDefinition, p.bound.recipe.Definition.ID, invalidDependencyPortMessage, nil)
	}
	call, t, finish, err := p.engine.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	return p.engine.resolve(call, p.bound.ports[index], resolution{
		task: t,
	})
}

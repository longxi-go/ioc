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
	"context"
	"path/filepath"
)

func renderGeneration(ctx context.Context, g *generation) error {
	for _, model := range g.local {
		if len(model.Recipes) == 0 {
			continue
		}
		data, err := emitPackage(model)
		if err != nil {
			return err
		}
		g.outputs[filepath.Join(model.Dir, generatedFileName)] = data
	}
	if g.options.Library {
		return nil
	}
	plan, err := emitPlan(g.entry, g.entry.Path, g.recipes, g.indexes)
	if err != nil {
		return err
	}
	path := filepath.Join(g.entry.Dir, generatedFileName)
	if adapter, ok := g.outputs[path]; ok {
		plan, err = combineSources(adapter, plan)
		if err != nil {
			return err
		}
	}
	g.outputs[path] = plan
	if err := renderHTTP(ctx, g); err != nil {
		return err
	}
	return nil
}

func publishGeneration(_ context.Context, g *generation) error {
	manifest, index, err := publish(publicationInput{
		Module:   g.module,
		Target:   g.target,
		Tags:     g.options.Tags,
		Packages: g.local,
		Recipes:  g.recipes,
		Outputs:  g.outputs,
	})
	if err != nil {
		return err
	}
	g.outputs[filepath.Join(g.module.Dir, publicationPath)] = manifest
	g.outputs[filepath.Join(g.module.Dir, indexSourcePath)] = index
	return nil
}

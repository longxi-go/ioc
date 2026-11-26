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
	"fmt"
	"path/filepath"
)

const httpGenerateFormat = "HTTP generate %s: %w"

func renderHTTP(ctx context.Context, g *generation) error {
	if g.options.HTTP == nil {
		return nil
	}
	var paths []string
	for _, model := range g.local {
		if hasHTTPPackage(model) && model.Path != g.entry.Path {
			paths = append(paths, model.Path)
		}
	}
	for _, model := range g.models {
		entry := model.Path == g.entry.Path
		if !hasHTTPPackage(model) && (!entry || len(paths) == 0) {
			continue
		}
		p := model.HTTP
		p.Entry = entry
		if entry {
			p.Packages = append([]string(nil), paths...)
		}
		data, err := g.options.HTTP.Generate(p)
		if err != nil {
			return fmt.Errorf(httpGenerateFormat, model.Path, err)
		}
		path := filepath.Join(model.Dir, generatedFileName)
		existing, ok := g.outputs[path]
		if !ok {
			existing, err = newEmitter(model.Path).finish(model.Name, model.Constraint)
			if err != nil {
				return fmt.Errorf(httpGenerateFormat, model.Path, err)
			}
		}
		names, resolveErr := httpImportNames(ctx, g, data, existing)
		if resolveErr != nil {
			return fmt.Errorf(httpGenerateFormat, model.Path, resolveErr)
		}
		data, err = combineNamedSources(data, existing, names)
		if err != nil {
			return fmt.Errorf(httpGenerateFormat, model.Path, err)
		}
		g.outputs[path] = data
	}
	return nil
}

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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const (
	generatedCodeFormat     = "generated code: %w"
	staleOutputFormat       = "generated output is stale: %s"
	staleOwnedAdapterFormat = "stale owned adapter: %s"
)

func validateGenerationOutput(ctx context.Context, g *generation) error {
	restoreHTTPOverlay(g)
	set, err := outputArtifacts(g.outputs, goLanguageVersionPrefix+g.module.GoVersion)
	if err != nil {
		return err
	}
	for path, data := range set.Overlay() {
		g.overlay[path] = data
	}
	for _, file := range set.Files() {
		g.report.Files = append(g.report.Files, file.Path)
	}
	recheck := append([]string{}, g.patterns...)
	if g.options.Library {
		recheck = append(recheck, relativePackagePrefix+indexPackageName)
	}
	if _, err := load(ctx, packageLoad{
		Dir:      g.dir,
		Tags:     g.options.Tags,
		Patterns: recheck,
		Overlay:  g.overlay,
	}); err != nil {
		return fmt.Errorf(generatedCodeFormat, err)
	}
	g.owned = map[string][]byte{}
	scopeDirs := map[string]bool{}
	for _, model := range g.models {
		scopeDirs[model.Dir] = true
	}
	if g.options.Library {
		scopeDirs[filepath.Join(g.module.Dir, indexPackageName)] = true
	}
	for path, data := range g.overlay {
		if scopeDirs[filepath.Dir(path)] {
			g.owned[path] = data
		}
	}
	g.validated = true
	return nil
}

func finishGeneration(ctx context.Context, g *generation) error {
	if g.options.Inspect {
		return nil
	}
	if g.options.Check {
		for _, path := range g.report.Files {
			current, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(current, g.outputs[path]) {
				return fmt.Errorf(staleOutputFormat, path)
			}
		}
		for path := range g.owned {
			if _, ok := g.outputs[path]; !ok {
				return fmt.Errorf(staleOwnedAdapterFormat, path)
			}
		}
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return commitOutputs(ctx, g.outputs, g.owned)
}

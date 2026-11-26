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
	"errors"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultPackagePattern = "./..."
	importPathSeparator   = "/"
	mainPackage           = "main"

	ambiguousApplicationRootsMessage    = "multiple application entries require explicit Roots to define entry closure"
	moduleDirectoryIsUnavailableMessage = "module directory is unavailable"
)

func prepareGeneration(ctx context.Context, g *generation) error {
	dir := g.options.Dir
	if dir == "" {
		dir = currentDirectory
	}
	var err error
	g.dir, err = filepath.Abs(dir)
	if err != nil {
		return err
	}
	g.module, err = loadModule(ctx, g.dir)
	if err != nil {
		return err
	}
	if g.module.Dir == "" {
		return errors.New(moduleDirectoryIsUnavailableMessage)
	}
	g.target, err = environment(ctx, g.dir)
	if err != nil {
		return err
	}
	g.overlay, err = ownedOverlay(g.module.Dir)
	if err != nil {
		return err
	}
	g.patterns = append([]string{}, g.options.Roots...)
	if len(g.patterns) == 0 {
		g.patterns = []string{defaultPackagePattern}
	}
	if !g.options.Library && g.options.Entry != "" {
		g.patterns = append(g.patterns, g.options.Entry)
	}
	return nil
}

func analyzeGeneration(ctx context.Context, g *generation) error {
	pkgs, err := load(ctx, packageLoad{
		Dir:      g.dir,
		Tags:     g.options.Tags,
		Patterns: g.patterns,
		Overlay:  g.overlay,
	})
	if err != nil {
		return err
	}
	if !g.options.Library && len(g.options.Roots) == 0 {
		entries := 0
		for _, pkg := range pkgs {
			if pkg.Module != nil && pkg.Module.Path == g.module.Path && pkg.Name == mainPackage {
				entries++
			}
		}
		if entries > 1 {
			return errors.New(ambiguousApplicationRootsMessage)
		}
	}
	for _, pkg := range pkgs {
		if pkg.Module == nil ||
			pkg.Module.Path != g.module.Path ||
			strings.HasSuffix(pkg.PkgPath, importPathSeparator+indexPackageName) {
			continue
		}
		if pkg.Name == mainPackage && g.options.Entry != "" {
			entries, err := load(ctx, packageLoad{
				Dir:      g.dir,
				Tags:     g.options.Tags,
				Patterns: []string{g.options.Entry},
				Overlay:  g.overlay,
			})
			if err != nil {
				return err
			}
			if len(entries) != 1 || pkg.PkgPath != entries[0].PkgPath {
				continue
			}
		}
		model, err := analyze(pkg)
		if err != nil {
			return err
		}
		model.Sources, err = sourceHashes(pkg)
		if err != nil {
			return err
		}
		model.Constraint, err = buildConstraint(pkg)
		if err != nil {
			return err
		}
		g.models = append(g.models, model)
		if len(model.Recipes) > 0 {
			g.local = append(g.local, model)
		}
	}
	sort.Slice(g.models, func(i, j int) bool {
		return g.models[i].Path < g.models[j].Path
	})
	sort.Slice(g.local, func(i, j int) bool {
		return g.local[i].Path < g.local[j].Path
	})
	return nil
}

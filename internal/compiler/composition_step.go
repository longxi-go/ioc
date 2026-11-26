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
	"sort"
	"strings"
)

const (
	ambiguousMainEntryFormat              = "application requires one explicit main entry (found %d); select Entry and Roots"
	libraryCannotPublishMainPackageFormat = "library cannot publish main package %s"
)

func composeGeneration(ctx context.Context, g *generation) error {
	var constraints []string
	for _, model := range g.local {
		if g.options.Library && model.Name == mainPackage {
			return fmt.Errorf(libraryCannotPublishMainPackageFormat, model.Path)
		}
		if model.Constraint != "" {
			constraints = append(constraints, constraintGroupOpen+model.Constraint+constraintGroupClose)
		}
		for _, recipe := range model.Recipes {
			published := publishedRecipe{
				Definition:  recipe.Definition,
				Conditional: len(recipe.Conditions) > 0,
				Conditions:  recipe.Conditions,
			}
			for _, listener := range recipe.Listeners {
				published.Listeners = append(published.Listeners, publishedListener{
					ID:        listener.ID,
					Event:     canonical(listener.Event),
					Source:    listener.Source,
					Order:     listener.Order,
					Lifecycle: listener.Lifecycle,
				})
			}
			for _, exposed := range recipe.Exposed {
				reference, err := describeType(exposed)
				if err != nil {
					return err
				}
				published.Types = append(published.Types, reference)
			}
			g.recipes = append(g.recipes, published)
		}
	}
	if g.options.Library {
		return nil
	}
	dependencies, err := selectedModules(ctx, g.module.Dir)
	if err != nil {
		return err
	}
	for _, dependency := range dependencies {
		if dependency.Path == g.module.Path {
			continue
		}
		manifest, exists, err := readPublication(dependency, g.target, g.options.Tags)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := validatePublicationBuild(ctx, publicationBuild{
			Dir:      g.dir,
			Tags:     g.options.Tags,
			Module:   dependency,
			Manifest: manifest,
		}); err != nil {
			return err
		}
		g.recipes = append(g.recipes, manifest.Recipes...)
		for _, pkg := range manifest.Packages {
			if pkg.Constraint != "" {
				constraints = append(constraints, constraintGroupOpen+pkg.Constraint+constraintGroupClose)
			}
		}
		g.indexes = append(g.indexes, dependency.Path+importPathSeparator+indexPackageName)
	}
	g.entry, err = chooseEntry(g.models)
	if err != nil {
		return err
	}
	for _, model := range g.local {
		if model.Path != g.entry.Path {
			g.indexes = append(g.indexes, model.Path)
		}
	}
	if g.entry.Constraint != "" {
		constraints = append(constraints, constraintGroupOpen+g.entry.Constraint+constraintGroupClose)
	}
	sort.Strings(constraints)
	g.entry.Constraint = strings.Join(constraints, buildConstraintConjunction)
	return nil
}

func validateGenerationGraph(_ context.Context, g *generation) error {
	graph, err := infrastructureGraph(g.recipes)
	if err != nil {
		return err
	}
	g.report.Edges, err = validateGraphClosure(graph, !g.options.Library)
	if err != nil {
		return err
	}
	for _, recipe := range g.recipes {
		g.report.Definitions = append(g.report.Definitions, recipe.Definition.Clone())
	}
	sort.Slice(g.report.Definitions, func(i, j int) bool {
		return g.report.Definitions[i].ID < g.report.Definitions[j].ID
	})
	return nil
}

func chooseEntry(models []*packageModel) (*packageModel, error) {
	var entries []*packageModel
	for _, model := range models {
		if model.Name == mainPackage {
			entries = append(entries, model)
		}
	}
	if len(entries) != 1 {
		return nil, fmt.Errorf(ambiguousMainEntryFormat, len(entries))
	}
	return entries[0], nil
}

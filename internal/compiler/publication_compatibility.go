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
)

const (
	missingBuildVariantPackageFormat     = "module %s incompatible build variant: a published adapter package is not selected"
	moduleIncompatibleBuildVariantFormat = "module %s incompatible build variant: %w"
	staleBuildVariantFormat              = "module %s incompatible build variant: static recipe or adapter constraints changed in %s; publish an adapter for the selected variant"
)

// adapterSemantics fingerprints the static type-checked recipe, independently of
// source headers, formatting versions and irrelevant build-environment flags.
// The generated body includes bindings, conditions, hooks, listeners and casts.
func adapterSemantics(model *packageModel) (string, error) {
	source, err := emitPackage(model)
	if err != nil {
		return "", err
	}
	return digest(source), nil
}

// validatePublicationBuild accepts equivalent consumer builds without changing
// published modules. Re-analysis checks selected types and the adapter's build
// constraints; raw GOOS/CGO/tag equality alone cannot establish compatibility.
type publicationBuild struct {
	Dir      string
	Tags     []string
	Module   module
	Manifest publication
}

func validatePublicationBuild(ctx context.Context, request publicationBuild) error {
	mod, manifest := request.Module, request.Manifest
	// Load from the consumer, so replacements and selected dependency versions
	// come from its module graph rather than the library author's go.mod.
	// Equal target flags are insufficient: dependency upgrades can change types.
	// Published adapters are checked as ordinary source: Go forbids overlays
	// beneath GOMODCACHE, and a consumer must not rewrite immutable dependencies.
	paths := make([]string, 0, len(manifest.Packages))
	for _, pkg := range manifest.Packages {
		paths = append(paths, pkg.Path)
	}
	packages, err := load(ctx, packageLoad{
		Dir:      request.Dir,
		Tags:     request.Tags,
		Patterns: paths,
	})
	if err != nil {
		return fmt.Errorf(moduleIncompatibleBuildVariantFormat, mod.Path, err)
	}
	expected := make(map[string]publishedPackage, len(manifest.Packages))
	for _, pkg := range manifest.Packages {
		expected[pkg.Path] = pkg
	}
	for _, pkg := range packages {
		published, exists := expected[pkg.PkgPath]
		if !exists {
			continue
		}
		model, err := analyze(pkg)
		if err != nil {
			return fmt.Errorf(moduleIncompatibleBuildVariantFormat, mod.Path, err)
		}
		model.Constraint, err = buildConstraint(pkg)
		if err != nil {
			return err
		}
		fingerprint, err := adapterSemantics(model)
		if err != nil {
			return err
		}
		if fingerprint != published.Semantics {
			return fmt.Errorf(staleBuildVariantFormat, mod.Path, pkg.PkgPath)
		}
		delete(expected, pkg.PkgPath)
	}
	if len(expected) > 0 {
		return fmt.Errorf(missingBuildVariantPackageFormat, mod.Path)
	}
	return nil
}

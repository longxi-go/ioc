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
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"

	"github.com/longxi-go/gosr/tools/pkg/packageload"
	"golang.org/x/tools/go/packages"
)

// Default import identifiers come from the target's selected packages, including
// its workspace, tags and isolated overlay. Path basenames are not package names.
func httpImportNames(ctx context.Context, g *generation, sources ...[]byte) (map[string]string, error) {
	paths := map[string]bool{}
	for _, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), adapterDiagnosticFilename, source, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, spec := range file.Imports {
			if spec.Name != nil {
				continue
			}
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			paths[path] = true
		}
	}
	names := map[string]string{}
	if len(paths) == 0 {
		return names, nil
	}
	patterns := make([]string, 0, len(paths))
	for path := range paths {
		patterns = append(patterns, path)
	}
	sort.Strings(patterns)
	cfg := packageload.Config(ctx, g.dir, g.overlay)
	cfg.Mode = packages.NeedName
	cfg.Env = commandEnvironment(ctx)
	if len(g.options.Tags) > 0 {
		cfg.BuildFlags = []string{buildTagsFlagPrefix + strings.Join(g.options.Tags, buildTagSeparator)}
	}
	pkgs, err := packageload.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if err := packageload.Errors(phaseTypeChecking, pkgs); err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		names[pkg.PkgPath] = pkg.Name
	}
	return names, nil
}

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
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"

	"github.com/longxi-go/gosr/pkg/codegen/imports"
)

const (
	fragmentImportAliasFormat      = "iocHTTPImport%d"
	fragmentImportPackageName      = "fragmentimport"
	fragmentDotAlias               = "."
	fragmentBindingPackage         = "ioc/fragment"
	unresolvedFragmentImportFormat = "unresolved default import %s"
)

type fragmentImporter map[string]string

func (names fragmentImporter) Import(path string) (*types.Package, error) {
	name := names[path]
	if name == "" {
		name = fragmentImportPackageName
	}
	pkg := types.NewPackage(path, name)
	pkg.MarkComplete()
	return pkg, nil
}

// fragmentImports preserves distinct aliases for the same path (legal Go), and
// only renames actual cross-path collisions. Fresh names avoid every identifier
// already present so moving an import cannot capture a local shadow variable.
func fragmentImports(fs *token.FileSet, files []*ast.File, names map[string]string) ([]importView, error) {
	reserved := map[string]bool{}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				reserved[id.Name] = true
			}
			return true
		})
	}
	bindings := map[*ast.File]*types.Info{}
	declared := map[string]bool{}
	for _, file := range files {
		info := &types.Info{
			Uses: map[*ast.Ident]types.Object{},
			Defs: map[*ast.Ident]types.Object{},
		}
		cfg := types.Config{
			Importer:                 fragmentImporter(names),
			DisableUnusedImportCheck: true,
			Error:                    func(error) {},
		}
		// Partial fragments may depend on private declarations and other generated
		// APIs. Resolve lexical bindings here; full overlay validation owns type errors.
		_, _ = cfg.Check(fragmentBindingPackage, fs, []*ast.File{file}, info)
		for id, obj := range info.Defs {
			if obj != nil {
				if _, imported := obj.(*types.PkgName); !imported {
					declared[id.Name] = true
				}
			}
		}
		bindings[file] = info
	}
	owners := map[string]string{}
	seen := map[importView]bool{}
	var result []importView
	for _, file := range files {
		info := bindings[file]
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			alias := names[path]
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			if alias == "" {
				return nil, fmt.Errorf(unresolvedFragmentImportFormat, path)
			}
			owner, occupied := owners[alias]
			if alias != imports.BlankAlias && alias != fragmentDotAlias && (occupied && owner != path || declared[alias] || types.Universe.Lookup(alias) != nil) {
				old := alias
				for i := 0; ; i++ {
					alias = fmt.Sprintf(fragmentImportAliasFormat, i)
					if !reserved[alias] {
						reserved[alias] = true
						break
					}
				}
				for id, obj := range info.Uses {
					imported, ok := obj.(*types.PkgName)
					if ok && imported.Imported().Path() == path && imported.Name() == old {
						id.Name = alias
					}
				}
			}
			if alias != imports.BlankAlias && alias != fragmentDotAlias {
				owners[alias] = path
			}
			view := importView{
				Path:  path,
				Alias: alias,
			}
			if !seen[view] {
				seen[view] = true
				result = append(result, view)
			}
		}
	}
	named := map[string]bool{}
	for _, view := range result {
		if view.Alias != imports.BlankAlias {
			named[view.Path] = true
		}
	}
	filtered := result[:0]
	for _, view := range result {
		if view.Alias != imports.BlankAlias || !named[view.Path] {
			filtered = append(filtered, view)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Path != filtered[j].Path {
			return filtered[i].Path < filtered[j].Path
		}
		return filtered[i].Alias < filtered[j].Alias
	})
	return filtered, nil
}

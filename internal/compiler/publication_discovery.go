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
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	sourceannotation "github.com/longxi-go/gosr/tools/pkg/annotation"
)

const (
	cgoEnabledValue = "1"
)

func publicationDeclarations(root, modulePath string, env target, tags []string) (map[string]bool, error) {
	result := map[string]bool{}
	buildContext := build.Default
	buildContext.GOOS = env.GOOS
	buildContext.GOARCH = env.GOARCH
	buildContext.CgoEnabled = env.CGO_ENABLED == cgoEnabledValue
	buildTags, err := effectiveTags(tags, env.GOFLAGS)
	if err != nil {
		return nil, err
	}
	buildContext.BuildTags = buildTags
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root &&
				(strings.HasPrefix(d.Name(), goFilenameDot) ||
					d.Name() == vendorDirectory ||
					d.Name() == testdataDirectory ||
					d.Name() == testsDirectory) {
				return filepath.SkipDir
			}
			if path != root {
				if _, err := os.Stat(filepath.Join(path, moduleFileName)); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !nonTestGoSource(d.Name()) {
			return nil
		}
		matches, err := buildContext.MatchFile(filepath.Dir(path), filepath.Base(path))
		if err != nil || !matches {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.HasPrefix(data, []byte(generatedHeader)) || !bytes.Contains(data, []byte(annotationPrefix)) {
			return nil
		}
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, path, data, parser.ParseComments)
		if err != nil {
			return err
		}
		if f.Name.Name == mainPackage {
			return nil
		}
		relative, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		pkgPath := modulePath
		if relative != currentDirectory {
			pkgPath += importPathSeparator + filepath.ToSlash(relative)
		}
		for _, decl := range f.Decls {
			switch v := decl.(type) {
			case *ast.FuncDecl:
				a, err := readMarkers(fs, v.Doc, declarationTarget)
				if err != nil {
					return err
				}
				if a.flag(markerBean) {
					result[pkgPath+definitionMemberSeparator+v.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range v.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					group := sourceannotation.TypeComments(v, ts)
					a, err := readMarkers(fs, group, declarationTarget)
					if err != nil {
						return err
					}
					if a.flag(markerComponent) {
						result[pkgPath+definitionMemberSeparator+ts.Name.Name] = true
					}
				}
			}
		}
		return nil
	})
	return result, err
}

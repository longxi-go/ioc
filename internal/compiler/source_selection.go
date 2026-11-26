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
	"crypto/sha256"
	"fmt"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	constraintGroupClose       = ")"
	constraintGroupOpen        = "("
	constraintNegatedGroupOpen = "!("
	digestFormat               = "%x"
)

// Constraint fragments retain their parentheses when combined so a package's
// OR expression cannot change the meaning of the surrounding conjunction.
const (
	buildConstraintPrefix      = "//go:build "
	buildConstraintConjunction = " && "
)

func ownedOverlay(root string) (map[string][]byte, error) {
	result := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root &&
				(strings.HasPrefix(d.Name(), goFilenameDot) || d.Name() == vendorDirectory || d.Name() == testdataDirectory) {
				return filepath.SkipDir
			}
			if path != root {
				if _, err := os.Stat(filepath.Join(path, moduleFileName)); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if ignoredGoFilename(d.Name()) || !strings.HasSuffix(path, goSourceExtension) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(b, []byte(generatedHeader)) {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, b, parser.PackageClauseOnly)
		if err != nil {
			return err
		}
		result[path] = []byte(packageClausePrefix + f.Name.Name + sourceLineSeparator)
		return nil
	})
	return result, err
}

func digest(data []byte) string {
	return fmt.Sprintf(digestFormat, sha256.Sum256(data))
}

func sourceHashes(pkg *packages.Package) (map[string]string, error) {
	result := map[string]string{}
	if len(pkg.GoFiles) == 0 {
		return result, nil
	}
	dir := filepath.Dir(pkg.GoFiles[0])
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !nonTestGoSource(entry.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if bytes.HasPrefix(b, []byte(generatedHeader)) {
			continue
		}
		result[entry.Name()] = digest(b)
	}
	return result, nil
}

func buildConstraint(pkg *packages.Package) (string, error) {
	if len(pkg.GoFiles) == 0 {
		return "", nil
	}
	active := map[string]bool{}
	for _, f := range pkg.GoFiles {
		active[filepath.Base(f)] = true
	}
	entries, err := os.ReadDir(filepath.Dir(pkg.GoFiles[0]))
	if err != nil {
		return "", err
	}
	var terms []string
	for _, entry := range entries {
		if entry.IsDir() || !nonTestGoSource(entry.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(filepath.Dir(pkg.GoFiles[0]), entry.Name()))
		if err != nil {
			return "", err
		}
		if bytes.HasPrefix(b, []byte(generatedHeader)) {
			continue
		}
		expression := ""
		for line := range strings.SplitSeq(string(b), sourceLineSeparator) {
			if strings.HasPrefix(line, buildConstraintPrefix) {
				expr, err := constraint.Parse(line)
				if err != nil {
					return "", err
				}
				expression = expr.String()
				break
			}
			if strings.HasPrefix(line, packageClausePrefix) {
				break
			}
		}
		suffix := filenameConstraint(entry.Name())
		if suffix != "" {
			if expression != "" {
				expression = constraintGroupOpen + expression + constraintGroupClose +
					buildConstraintConjunction + constraintGroupOpen + suffix + constraintGroupClose
			} else {
				expression = suffix
			}
		}
		if expression == "" {
			continue
		}
		if active[entry.Name()] {
			terms = append(terms, constraintGroupOpen+expression+constraintGroupClose)
		} else {
			expr, err := constraint.Parse(buildConstraintPrefix + expression)
			if err != nil {
				return "", err
			}
			if not, ok := expr.(*constraint.NotExpr); ok {
				terms = append(terms, constraintGroupOpen+not.X.String()+constraintGroupClose)
			} else {
				terms = append(terms, constraintNegatedGroupOpen+expression+constraintGroupClose)
			}
		}
	}
	sort.Strings(terms)
	return strings.Join(terms, buildConstraintConjunction), nil
}

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
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

const isolatedModuleAliasFormat = "module%d"

// inspectionWorkspace owns only the private view for one read-only invocation.
// Source paths are retained so diagnostics can be translated back after tooling.
type inspectionWorkspace struct {
	originalDir string
	root        module
	temporary   string
	workPath    string
	work        *modfile.WorkFile
	mirrors     map[string]string
}

func (w *inspectionWorkspace) discover(ctx context.Context) error {
	workBytes, err := goOutput(ctx, w.originalDir, goEnvCommand, goWorkKey)
	if err != nil {
		return err
	}
	w.workPath = strings.TrimSpace(string(workBytes))
	workPath := w.workPath
	roots := []string{w.root.Dir}
	var work *modfile.WorkFile
	if workPath != "" && workPath != goWorkOff {
		data, err := os.ReadFile(workPath)
		if err != nil {
			return err
		}
		work, err = modfile.ParseWork(workPath, data, nil)
		if err != nil {
			return err
		}
		for _, use := range work.Use {
			path := use.Path
			if !filepath.IsAbs(path) {
				path = filepath.Join(filepath.Dir(workPath), path)
			}
			path = filepath.Clean(path)
			if path != w.root.Dir {
				roots = append(roots, path)
			}
		}
	}
	w.work = work
	w.mirrors = map[string]string{}
	for i, path := range roots {
		w.mirrors[path] = filepath.Join(w.temporary, fmt.Sprintf(isolatedModuleAliasFormat, i))
	}
	return nil
}

func (w *inspectionWorkspace) copyModules() error {
	for original, mirror := range w.mirrors {
		if err := w.copyModule(original, mirror); err != nil {
			return err
		}
	}
	return nil
}

func (w *inspectionWorkspace) copyModule(original, mirror string) error {
	if err := mirrorSources(original, mirror); err != nil {
		return err
	}
	modPath := filepath.Join(original, moduleFileName)
	if original == w.root.Dir && w.root.GoMod != "" {
		modPath = w.root.GoMod
	}
	data, err := os.ReadFile(modPath)
	if err != nil {
		return err
	}
	mod, err := modfile.Parse(moduleFileName, data, nil)
	if err != nil {
		return err
	}
	for _, replace := range mod.Replace {
		if replace.New.Version != "" {
			continue
		}
		replacement := replace.New.Path
		if !filepath.IsAbs(replacement) {
			replacement = filepath.Join(original, replacement)
		}
		if shadow, ok := w.mirrors[filepath.Clean(replacement)]; ok {
			replacement = shadow
		}
		if err := mod.AddReplace(replace.Old.Path, replace.Old.Version, replacement, ""); err != nil {
			return err
		}
	}
	data, err = mod.Format()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(mirror, moduleFileName), data, inspectionFilePermissions); err != nil {
		return err
	}
	return nil
}

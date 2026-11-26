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

package testmodule

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// WorkspaceEnv opts temporary test consumers into local workspace dependencies.
// Production module files and default published-dependency tests stay unchanged.
const WorkspaceEnv = "IOC_TEST_WORKSPACE"

// UseWorkspace applies the explicit test workspace to a copied source module.
// Without WorkspaceEnv it leaves the copy's published dependency graph intact.
func UseWorkspace(dir string) error {
	if os.Getenv(WorkspaceEnv) == "" {
		return nil
	}
	path := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	module, err := modfile.Parse(path, data, nil)
	if err != nil {
		return err
	}
	if module.Module == nil {
		return fmt.Errorf("copied source module lacks a module directive")
	}
	if err := workspaceDependencies(module, module.Module.Mod.Path); err != nil {
		return err
	}
	data, err = module.Format()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func workspaceDependencies(consumer *modfile.File, sourcePath string) error {
	path := os.Getenv(WorkspaceEnv)
	if path == "" {
		return nil
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s must name an absolute go.work path", WorkspaceEnv)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	work, err := modfile.ParseWork(path, data, nil)
	if err != nil {
		return err
	}
	required := make(map[string]bool, len(consumer.Require))
	for _, requirement := range consumer.Require {
		required[requirement.Mod.Path] = true
	}
	for _, use := range work.Use {
		dir := use.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(filepath.Dir(path), dir)
		}
		modPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err != nil {
			return err
		}
		module, err := modfile.Parse(modPath, data, nil)
		if err != nil {
			return err
		}
		if module.Module == nil {
			return fmt.Errorf("workspace module %s lacks a module directive", modPath)
		}
		name := module.Module.Mod.Path
		if name != sourcePath && required[name] {
			if err := consumer.AddReplace(name, "", filepath.Clean(dir), ""); err != nil {
				return err
			}
		}
	}
	return nil
}

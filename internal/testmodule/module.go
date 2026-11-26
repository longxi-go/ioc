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

// Write initializes an existing consumer directory with the source module's
// dependency requirements and checksums. By default only the source module is
// replaced; dependencies retain their published identities.
// WorkspaceEnv explicitly opts isolated test consumers into local use entries;
// only temporary module files receive replacements, never production go.mod.
// Copying its complete graph also prepares imports introduced during generation,
// before those imports appear in the consumer's handwritten source.
func Write(dir, modulePath, sourceRoot string) error {
	root, err := filepath.Abs(sourceRoot)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	source, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return err
	}
	if source.Module == nil || source.Go == nil {
		return fmt.Errorf("source module must declare its module path and Go version")
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		return err
	}
	var consumer modfile.File
	if err := consumer.AddModuleStmt(modulePath); err != nil {
		return err
	}
	if err := consumer.AddGoStmt(source.Go.Version); err != nil {
		return err
	}
	if err := consumer.AddRequire(source.Module.Mod.Path, "v0.0.0"); err != nil {
		return err
	}
	for _, requirement := range source.Require {
		consumer.AddNewRequire(requirement.Mod.Path, requirement.Mod.Version, requirement.Indirect)
	}
	if err := consumer.AddReplace(source.Module.Mod.Path, "", root, ""); err != nil {
		return err
	}
	if err := workspaceDependencies(&consumer, source.Module.Mod.Path); err != nil {
		return err
	}
	data, err = consumer.Format()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), data, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644)
}

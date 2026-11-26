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
	"os"
	"path/filepath"
)

const (
	inspectionFilePermissions      = 0o644
	inspectionDirectoryPermissions = 0o755
	gitDirectory                   = ".git"
	localTemporaryDirectory        = ".tmp"
	nodeModulesDirectory           = "node_modules"
)

func excludedInspectionDirectory(name string) bool {
	switch name {
	case gitDirectory, localTemporaryDirectory, nodeModulesDirectory:
		return true
	default:
		return false
	}
}

func mirrorSources(original, mirror string) error {
	return filepath.WalkDir(original, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(original, path)
		if err != nil {
			return err
		}
		target := filepath.Join(mirror, relative)
		if d.IsDir() {
			if path != original && excludedInspectionDirectory(d.Name()) {
				return filepath.SkipDir
			}
			if d.Name() == vendorDirectory {
				if err := os.Symlink(path, target); err != nil {
					return err
				}
				return filepath.SkipDir
			}
			return os.MkdirAll(target, inspectionDirectoryPermissions)
		}
		if relative == moduleFileName || relative == moduleSumFileName {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, inspectionFilePermissions)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return os.Symlink(path, target)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

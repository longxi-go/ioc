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

	"golang.org/x/mod/modfile"
)

const workspaceSumSuffix = ".sum"

func (w *inspectionWorkspace) writeWorkspace() (string, error) {
	if w.work == nil {
		return goWorkOff, nil
	}
	uses := append([]*modfile.Use{}, w.work.Use...)
	for _, use := range uses {
		path := use.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(w.workPath), path)
		}
		shadow := w.mirrors[filepath.Clean(path)]
		if err := w.work.DropUse(use.Path); err != nil {
			return "", err
		}
		if err := w.work.AddUse(shadow, use.ModulePath); err != nil {
			return "", err
		}
	}
	for _, replace := range w.work.Replace {
		replacement := replace.New.Path
		if replace.New.Version == "" {
			if !filepath.IsAbs(replacement) {
				replacement = filepath.Join(filepath.Dir(w.workPath), replacement)
			}
			if shadow, ok := w.mirrors[filepath.Clean(replacement)]; ok {
				replacement = shadow
			}
		}
		if err := w.work.AddReplace(replace.Old.Path, replace.Old.Version, replacement, replace.New.Version); err != nil {
			return "", err
		}
	}

	if _, err := os.Stat(filepath.Join(filepath.Dir(w.workPath), vendorDirectory)); err == nil {
		if err := os.Symlink(filepath.Join(filepath.Dir(w.workPath), vendorDirectory), filepath.Join(w.temporary, vendorDirectory)); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	shadowWorkPath := filepath.Join(w.temporary, workspaceFileName)
	if err := os.WriteFile(shadowWorkPath, modfile.Format(w.work.Syntax), inspectionFilePermissions); err != nil {
		return "", err
	}
	sum, err := os.ReadFile(w.workPath + workspaceSumSuffix)
	if err == nil {
		if err := os.WriteFile(shadowWorkPath+workspaceSumSuffix, sum, inspectionFilePermissions); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return shadowWorkPath, nil
}

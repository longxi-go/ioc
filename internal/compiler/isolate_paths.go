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
	"errors"
	"path/filepath"
)

const entryMustBelongToTheOwnedModuleMessage = "entry must belong to the owned module"

func (w *inspectionWorkspace) options(options Options) (Options, error) {
	relative, err := filepath.Rel(w.root.Dir, w.originalDir)
	if err != nil {
		return Options{}, err
	}
	options.Dir = filepath.Join(w.mirrors[w.root.Dir], relative)
	if filepath.IsAbs(options.Entry) {
		rel, err := filepath.Rel(w.root.Dir, options.Entry)
		if err != nil || !safeRelative(rel) {
			return Options{}, errors.New(entryMustBelongToTheOwnedModuleMessage)
		}
		options.Entry = filepath.Join(w.mirrors[w.root.Dir], rel)
	}
	options.isolated = true
	return options, nil
}

func (w *inspectionWorkspace) restore(report *Report) {
	if report != nil {
		for i, path := range report.Files {
			for original, mirror := range w.mirrors {
				rel, err := filepath.Rel(mirror, path)
				if err == nil && safeRelative(rel) {
					report.Files[i] = filepath.Join(original, rel)
					break
				}
			}
		}
	}
}

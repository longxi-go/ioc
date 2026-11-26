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
	"path/filepath"

	"github.com/longxi-go/gosr/pkg/codegen/artifact"
)

// outputArtifacts snapshots Go and publication assets without moving the recipe
// ABI or manifest into shared infrastructure. Map ownership remains invocation-local.
func outputArtifacts(outputs map[string][]byte, goVersion string) (*artifact.Set, error) {
	files := make([]artifact.File, 0, len(outputs))
	for path, data := range outputs {
		kind := artifact.Data
		if filepath.Ext(path) == goSourceExtension {
			kind = artifact.Go
		}
		files = append(files, artifact.File{
			Path:      path,
			Content:   data,
			Kind:      kind,
			GoVersion: goVersion,
		})
	}
	return artifact.New(files...)
}

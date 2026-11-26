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
	"context"
	"path/filepath"

	fileoutput "github.com/longxi-go/gosr/pkg/codegen/output"
)

// commitOutputs supplies IoC artifact ownership and explicit stale selection;
// staging, filesystem preflight and cleanup are owned by the shared committer.
func commitOutputs(ctx context.Context, outputs, overlay map[string][]byte) error {
	set, err := outputArtifacts(outputs, "")
	if err != nil {
		return err
	}
	var stale []string
	for path := range overlay {
		if _, ok := outputs[path]; !ok {
			stale = append(stale, path)
		}
	}
	return fileoutput.Commit(ctx, set, fileoutput.Options{
		Owns:  ownsOutput,
		Stale: stale,
	})
}

func ownsOutput(path string, current []byte) bool {
	return filepath.Ext(path) != goSourceExtension || bytes.HasPrefix(current, []byte(generatedHeader))
}

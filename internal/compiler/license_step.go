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

	"github.com/longxi-go/golayout/pkg/header"
)

const (
	qualityDirectory = "codequality"
	sourceHeaderFile = "HEADER"

	sourceHeaderErrorFormat = "source header %s: %w"
)

// Notices belong to the target module, never to the compiler distribution.
// This step runs before formatting and before library artifact hashes are made.
func licenseGeneration(ctx context.Context, g *generation) error {
	notice, err := os.ReadFile(filepath.Join(g.module.Dir, qualityDirectory, sourceHeaderFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for path, source := range g.outputs {
		if filepath.Ext(path) != goSourceExtension {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		updated, err := header.Source(source, notice)
		if err != nil {
			return fmt.Errorf(sourceHeaderErrorFormat, path, err)
		}
		g.outputs[path] = updated
	}
	return nil
}

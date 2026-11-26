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
	"errors"
	"os"
	"path/filepath"
)

const inspectionDirectoryPattern = "ioc-inspect-*"

// isolateReadOnly gives Go tooling private module metadata and copied source
// views. This avoids restoring files after a tool writes them, which would race
// user edits and would make read-only commands observable as writers.
func isolateReadOnly(ctx context.Context, options Options) (report *Report, resultErr error) {
	originalDir := options.Dir
	if originalDir == "" {
		originalDir = currentDirectory
	}
	originalDir, err := filepath.Abs(originalDir)
	if err != nil {
		return nil, err
	}
	root, err := loadModule(ctx, originalDir)
	if err != nil {
		return nil, err
	}
	temporary, err := os.MkdirTemp("", inspectionDirectoryPattern)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, os.RemoveAll(temporary))
	}()

	workspace := inspectionWorkspace{
		originalDir: originalDir,
		root:        root,
		temporary:   temporary,
	}
	if err := workspace.discover(ctx); err != nil {
		return nil, err
	}
	if err := workspace.copyModules(); err != nil {
		return nil, err
	}
	env, err := inspectionEnvironment(ctx, originalDir)
	if err != nil {
		return nil, err
	}
	workPath, err := workspace.writeWorkspace()
	if err != nil {
		return nil, err
	}
	env = replaceEnvironment(env, goWorkKey, workPath)
	options, err = workspace.options(options)
	if err != nil {
		return nil, err
	}
	report, resultErr = Run(context.WithValue(ctx, environmentKey{}, env), options)
	workspace.restore(report)
	return report, resultErr
}

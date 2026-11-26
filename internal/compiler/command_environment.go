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
	"os"
	"strings"
)

const environmentAssignmentSeparator = "="

type environmentKey struct{}

func commandEnvironment(ctx context.Context) []string {
	if env, ok := ctx.Value(environmentKey{}).([]string); ok {
		return env
	}
	return os.Environ()
}

func replaceEnvironment(env []string, key, value string) []string {
	var result []string
	for _, item := range env {
		if !strings.HasPrefix(item, key+environmentAssignmentSeparator) {
			result = append(result, item)
		}
	}
	return append(result, key+environmentAssignmentSeparator+value)
}

// inspectionEnvironment removes only the original module-file override. The
// mirrored workspace still receives the caller's other Go flags and environment.
func inspectionEnvironment(ctx context.Context, dir string) ([]string, error) {
	env := commandEnvironment(ctx)
	flags, err := goOutput(ctx, dir, goEnvCommand, goFlagsKey)
	if err != nil {
		return nil, err
	}
	var safeFlags []string
	words, err := splitFlagWords(string(flags))
	if err != nil {
		return nil, err
	}
	for _, flag := range words {
		if !strings.HasPrefix(flag, moduleFileFlagPrefix) {
			safeFlags = append(safeFlags, flag)
		}
	}
	serialized, err := joinFlagWords(safeFlags)
	if err != nil {
		return nil, err
	}
	env = replaceEnvironment(env, goFlagsKey, serialized)
	return env, nil
}

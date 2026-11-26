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

	"github.com/longxi-go/golayout/pkg/format"
	"github.com/longxi-go/gosr/pkg/codegen/artifact"
)

const (
	formatOutputErrorFormat = "format %s: %w"
)

func formatGeneration(ctx context.Context, g *generation) error {
	set, err := outputArtifacts(g.outputs, goLanguageVersionPrefix+g.module.GoVersion)
	if err != nil {
		return err
	}
	for _, file := range set.Files() {
		if file.Kind != artifact.Go {
			continue
		}
		path := file.Path
		if err := ctx.Err(); err != nil {
			return err
		}
		// Type analysis owns import selection. Formatting must not resolve
		// packages or remove explicit aliases and side-effect imports.
		formatted, err := format.Source(g.outputs[path], format.Options{
			LangVersion: file.GoVersion,
		})
		if err != nil {
			return fmt.Errorf(formatOutputErrorFormat, path, err)
		}
		g.outputs[path] = formatted
	}
	return nil
}

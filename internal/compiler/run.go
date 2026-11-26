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

	"github.com/longxi-go/gosr/pkg/pipeline"
)

// Run compiles declarations through explicit steps. The isolated read-only path
// protects module/source metadata before the pipeline can inspect an application.
func Run(ctx context.Context, options Options) (*Report, error) {
	if (options.Check || options.Inspect) && !options.isolated {
		return isolateReadOnly(ctx, options)
	}
	g := &generation{
		options: options,
		report:  &Report{},
		outputs: map[string][]byte{},
	}
	if err := pipeline.Run(ctx, g, generationSteps(options)...); err != nil {
		if g.validated && options.Check {
			return g.report, err
		}
		return nil, err
	}
	return g.report, nil
}

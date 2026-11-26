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

import "github.com/longxi-go/gosr/pkg/pipeline"

const (
	stagePrepare            = "prepare"
	stageCompose            = "compose"
	stageRender             = "render"
	stageFinishOutput       = "finish output"
	stageFormatGo           = "format Go"
	stageFormatIndex        = "format publication index"
	stagePublicationHeaders = "apply publication source headers"
	stagePublishMetadata    = "publish library metadata"
	stageSourceHeaders      = "apply source headers"
	stageValidateGraph      = "validate graph"
	stageValidateOverlay    = "validate generated overlay"
)

func generationSteps(options Options) []pipeline.Step[*generation] {
	steps := []pipeline.Step[*generation]{
		pipeline.NewStep(stagePrepare, prepareGeneration),
		pipeline.NewStep(stageAnalyze, analyzeGeneration),
		pipeline.NewStep(stageCompose, composeGeneration),
		pipeline.NewStep(stageValidateGraph, validateGenerationGraph),
		pipeline.NewStep(stageRender, renderGeneration),
		pipeline.NewStep(stageSourceHeaders, licenseGeneration),
		pipeline.NewStep(stageFormatGo, formatGeneration),
	}
	if options.Library {
		// The manifest hashes final adapter bytes. Its generated index is a new
		// Go artifact and passes through the same formatter before validation.
		steps = append(steps,
			pipeline.NewStep(stagePublishMetadata, publishGeneration),
			pipeline.NewStep(stagePublicationHeaders, licenseGeneration),
			pipeline.NewStep(stageFormatIndex, formatGeneration),
		)
	}
	return append(steps,
		pipeline.NewStep(stageValidateOverlay, validateGenerationOutput),
		pipeline.NewStep(stageFinishOutput, finishGeneration),
	)
}

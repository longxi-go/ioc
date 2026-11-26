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
	"slices"
	"testing"
)

func TestApplicationGenerationStageOrder(t *testing.T) {
	assertGenerationStageOrder(t, Options{}, []string{
		"prepare",
		"analyze",
		"compose",
		"validate graph",
		"render",
		"apply source headers",
		"format Go",
		"validate generated overlay",
		"finish output",
	})
}

func TestLibraryPublicationRunsBetweenFormattingAndValidation(t *testing.T) {
	assertGenerationStageOrder(t, Options{
		Library: true,
	}, []string{
		"prepare",
		"analyze",
		"compose",
		"validate graph",
		"render",
		"apply source headers",
		"format Go",
		"publish library metadata",
		"apply publication source headers",
		"format publication index",
		"validate generated overlay",
		"finish output",
	})
}

func assertGenerationStageOrder(t *testing.T, options Options, want []string) {
	t.Helper()
	var names []string
	for _, stage := range generationSteps(options) {
		names = append(names, stage.Name())
	}
	if !slices.Equal(names, want) {
		t.Fatalf("stage order = %v, want %v", names, want)
	}
}

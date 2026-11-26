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

package bean

// Shared validation text keeps authored recipes, generated adapters and runtime
// binding diagnostics consistent. Error codes remain the classification API.
const (
	// MissingFactoryMessage identifies a registration without a creation adapter.
	MissingFactoryMessage = "missing factory"
	// EagerSingletonStartMessage identifies an invalid application Start owner.
	EagerSingletonStartMessage = "application Start requires eager singleton"
	// InvalidPrerequisiteMessage identifies a self or duplicate prerequisite.
	InvalidPrerequisiteMessage = "self or duplicate initialization prerequisite "
)

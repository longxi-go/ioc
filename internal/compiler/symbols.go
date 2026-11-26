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

// These spellings belong to generated adapters and static type recognition.
// Keep them separate from Go syntax, annotation names and runtime Bean identity.
const (
	publisherType        = "Publisher"
	environmentType      = "Environment"
	contextRefreshedType = "ContextRefreshed"
	contextClosingType   = "ContextClosing"
	contextPackage       = "context"
	contextType          = "Context"
	contractsAlias       = "gs"
	beanPackage          = runtimeModule + "/pkg/bean"
	beanAlias            = "bean"
	catalogPackage       = runtimeModule + "/pkg/catalog"
	catalogAlias         = "catalog"
	providerType         = "Provider"
	optionalType         = "Optional"
	eventType            = "Event"
	eventMarkerMethod    = "EventMarker"
	onEventMethod        = "OnEvent"
)

type lifecycleHook string

const (
	initHook    lifecycleHook = "Init"
	startHook   lifecycleHook = "Start"
	stopHook    lifecycleHook = "Stop"
	destroyHook lifecycleHook = "Destroy"
)

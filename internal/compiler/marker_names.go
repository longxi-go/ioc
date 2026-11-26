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

// Marker names align controller-tools registration, semantic analysis and
// dependency scans. They belong to this compiler rather than portable IoC.
const (
	markerNamespace     = "ioc:"
	annotationPrefix    = "+" + markerNamespace
	markerProfile       = "profile"
	markerParam         = "param"
	markerBean          = "bean"
	markerComponent     = "component"
	markerName          = "name"
	markerScope         = "scope"
	markerQualifier     = "qualifier"
	markerExpose        = "expose"
	markerAlias         = "alias"
	markerPrimary       = "primary"
	markerLazy          = "lazy"
	markerOnProperty    = "conditional:on-property"
	markerOnMissingBean = "conditional:on-missing-bean"
	markerOnBean        = "conditional:on-bean"
	markerOrder         = "order"
	markerDependsOn     = "depends-on"
	markerListener      = "listener"
	markerListen        = "listen"
	markerInject        = "inject"
)

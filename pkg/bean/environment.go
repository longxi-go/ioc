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

import gs "github.com/longxi-go/gosr/pkg/ioc"

const (
	// EnvironmentID identifies initialized per-context infrastructure, never a catalog contribution.
	EnvironmentID = "ioc.infrastructure/environment"
	// EnvironmentName reserves a stable lookup name for the configured owner.
	EnvironmentName = "iocEnvironment"
	// EnvironmentContract is the canonical read capability exposed to generated adapters.
	EnvironmentContract gs.Contract = "github.com/longxi-go/gosr/pkg/environment.Environment"
)

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

package context

import env "github.com/longxi-go/gosr/pkg/environment"

// SnapshotEnvironment supplies an immutable selection view while retaining the
// same potentially live Environment owner for application and Bean reads.
// Implementations must satisfy both Gosr concurrency and snapshot contracts.
type SnapshotEnvironment interface {
	env.Environment
	env.SnapshotProvider
}

// Environment returns the initialized owner shared with injected Environment
// dependencies. Conditions continue using the snapshot captured at construction.
func (a *Application) Environment() env.Environment {
	return a.environment
}

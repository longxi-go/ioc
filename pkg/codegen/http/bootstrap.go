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

package http

// Bootstrapper optionally supplies temporary typed declarations before source
// analysis, allowing application code to reference generated entry points on its
// first generation. Only Package.Path and Package.Name are populated. The compiler
// never writes or executes this source and removes it before final validation;
// Generate must provide the real declarations. An empty result adds no overlay.
type Bootstrapper interface {
	Bootstrap(Package) ([]byte, error)
}

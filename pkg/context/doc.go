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

// Package context is the public application facade over the private ownership
// engine. Generated plans supply typed query dispatch, keeping reflection and
// type discovery out of runtime lookup. Applications snapshot catalogs and own
// independent instances. Construction prepares Nemo or a supplied Environment
// before ordinary factories, pinning a selection snapshot while sharing the live
// owner through Gosr. Start activates once and Close remains joinable after
// cancellation. ConditionReport evaluates the same metadata selection as Start
// without executing factories, and remains usable after failed startup.
// Explicit scopes and leases express shorter resource lifetimes.
// Application also implements event.Publisher; the injected default uses the
// same local synchronous dispatcher. Publications pin effective scopes and stay
// tracked until callbacks return, so shutdown cannot destroy borrowed objects.
package context

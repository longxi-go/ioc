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

// Package compiler implements source-only IoC analysis and deterministic adapter
// publication. It never executes discovered factories or writes dependencies.
// Published metadata is treated as untrusted input and checked against sources.
// The compiler owns its ordered stages and invocation state; the shared gosr
// pipeline executes those stages. Formatting precedes library hashing, so
// metadata describes the exact adapter bytes that will be written.
// Shared codegen packages own imports, artifact snapshots, diagnostic causes,
// strict templates and filesystem commit. The separate gosr/tools module owns
// copied package-loading inputs and controller-tools comment adaptation.
// Domain schemas, models and module/discovery policy remain in this package.
// Dependency syntax has a finite grammar; named condition attributes resolve
// source-scoped types before emitting independent existence predicates.
package compiler

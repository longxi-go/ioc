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

// Package testmodule prepares temporary consumer modules for generation tests.
// It shares the module under test's real dependency graph and checksums while
// replacing only that module with its source checkout. This keeps isolated
// fixture resolution independent of sibling repositories and go.work selection.
// WorkspaceEnv explicitly selects a local go.work for source development;
// its required use entries become replacements only in temporary consumers.
// That mode verifies local contracts, not published dependency compatibility.
//
// This package is test infrastructure, not part of the generation pipeline.
// Callers own source files, Go commands, and any deliberate metadata corruption
// needed to exercise read-only or invalid-module behavior.
package testmodule

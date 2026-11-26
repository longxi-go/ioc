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

// Package eventdispatch prepares and invokes immutable generated listener
// declarations. The container supplies instance resolution and phase admission;
// this package owns matching, sampled precedence and invocation-chain policy.
// It does not create scopes, discover packages or own application lifecycles.
// No user callback runs under a dispatcher lock or during descriptor registration.
package eventdispatch

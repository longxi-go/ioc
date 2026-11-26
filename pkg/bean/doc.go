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

// Package bean is the static protocol between generated adapters and the managed
// runtime. Recipes contain executable Go callbacks and copied gosr metadata;
// contributing recipes never allocates application objects. Ports are already
// bound dependencies, avoiding runtime reflection or string-based field wiring.
// Listener adapters belong to their owner recipe; matching and invocation never
// execute during contribution. Registration.Clone owns mutable descriptor and
// condition metadata. Condition reports expose detached predicate observations
// without executable callbacks, expanded property keys or property values.
// SchemaVersion identifies incompatible generated protocols.
package bean

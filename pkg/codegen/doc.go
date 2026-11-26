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

// Package codegen provides the source-comment IoC compiler. Libraries publish
// adapters and a module index; applications bind those indexes into one typed
// query plan. Compilation type-checks sources and generated overlays without
// executing user code. Run owns only files marked as ioc-generated.
//
// Constructor parameters can select candidates explicitly with repeatable
// +ioc:param:parameter=read,name=readDB comments (qualifiers={sql,read} is
// optional). Parameters never infer bean names from Go parameter names.
// Injection accepts T, Optional[T], []T and an outer Provider wrapping each.
// Nested cardinality operators and inner Providers are generation errors.
// Named collections, maps and arrays remain exact contracts.
//
// Bean conditions use +ioc:conditional:on-bean or
// +ioc:conditional:on-missing-bean, with optional type and name attributes.
// Type expressions resolve in the declaration's file scope; an invalid type
// never falls back to a name. Each attribute is an independent requirement;
// omitting both infers the declared result type. Conditions select definitions
// before construction; they do not establish instance readiness.
// Property conditions use structured name/prefix/havingValue/matchIfMissing
// attributes. Properties and profiles read one startup Environment snapshot.
// The compiler retains expressions without reading the developer environment.
package codegen

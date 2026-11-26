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

import (
	"go/types"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

// Adapter emits a complete Go source fragment for one owned package. The compiler
// combines, formats and type-checks it before publishing; adapters must not write files.
type Adapter interface {
	Generate(Package) ([]byte, error)
}

// Route retains typed signatures so a transport can validate its native handler
// contract without making the common compiler depend on that transport.
type Route struct {
	Method, Path, Handler string
	Signature             *types.Signature
	Source                gs.Source
}

// Controller associates local pointer syntax with the exact managed definition.
// Its handlers resolve the definition at request time, preserving scope ownership.
type Controller struct {
	Definition gs.BeanDefinition
	Type       string
	Routes     []Route
}

// Package describes one owned source package. Entry aggregates only the other
// HTTP Packages; local declarations remain available through the package binder.
type Package struct {
	Path, Name  string
	Controllers []Controller
	Handlers    []Route
	Entry       bool
	Packages    []string
}

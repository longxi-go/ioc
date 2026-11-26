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

package repository

import (
	"context"

	"github.com/longxi-go/ioc/examples/hello/contracts"
)

var _ contracts.Repository = (*Database)(nil)

// Database has private state; generated same-package code needs no reflection.
// +ioc:component
// +ioc:expose=contracts.Repository
type Database struct {
	message string
}

// Init prepares the allocation before a consumer can observe it.
func (d *Database) Init(context.Context) error {
	d.message = "hello from static IoC"
	return nil
}

// Message is business behavior with no dependency on a container.
func (d *Database) Message() string {
	return d.message
}

// Destroy releases the resource even after a partial startup failure.
func (d *Database) Destroy(context.Context) error {
	d.message = "closed"
	return nil
}

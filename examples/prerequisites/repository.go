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

package main

import (
	"fmt"
	"slices"
)

// Repository requires completed setup without retaining the migration object.
type Repository struct {
	trace *Trace
}

// NewRepository verifies that a named prerequisite completed Init first.
//
// +ioc:bean
// +ioc:depends-on=databaseMigration
func NewRepository(trace *Trace) (*Repository, error) {
	if !slices.Equal(trace.Entries, []string{"migration initialized"}) {
		return nil, fmt.Errorf("migration has not completed: %v", trace.Entries)
	}
	trace.Entries = append(trace.Entries, "repository constructed")
	return &Repository{
		trace: trace,
	}, nil
}

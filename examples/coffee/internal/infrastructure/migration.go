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

package infrastructure

import (
	"context"
	_ "embed"
)

//go:embed schema.sql
var schema string

// Migration prepares tables before a Repository can be constructed.
type Migration struct {
	database *Database
}

// NewMigration is a named prerequisite; a repository does not retain it.
// +ioc:bean:name=databaseMigration
func NewMigration(database *Database) *Migration {
	return &Migration{
		database: database,
	}
}

func (m *Migration) Init(ctx context.Context) error {
	_, err := m.database.SQL().ExecContext(ctx, schema)
	return err
}

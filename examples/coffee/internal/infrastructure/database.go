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
	"database/sql"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	_ "modernc.org/sqlite"
)

const (
	sqliteDriver         = "sqlite"
	fileScheme           = "file"
	pragmaParameter      = "_pragma"
	busyPragma           = "busy_timeout(5000)"
	foreignKeysPragma    = "foreign_keys(1)"
	transactionParameter = "_txlock"
	immediateTransaction = "immediate"
	journalStatement     = "PRAGMA journal_mode=WAL"
	maxConnections       = 4
)

// Database owns the connection pool; consumers use SQL without closing the shared pool.
type Database struct {
	settings  *configuration.Settings
	pool      *sql.DB
	closeOnce sync.Once
	closeErr  error
}

// NewDatabase defers opening until the container runs Init.
// +ioc:bean
func NewDatabase(settings *configuration.Settings) *Database {
	return &Database{
		settings: settings,
	}
}

// Init applies connection-local settings through the driver's DSN for every connection.
func (d *Database) Init(ctx context.Context) error {
	absolute, err := filepath.Abs(d.settings.Database.Path)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Add(pragmaParameter, busyPragma)
	query.Add(pragmaParameter, foreignKeysPragma)
	query.Set(transactionParameter, immediateTransaction)
	uri := url.URL{
		Scheme:   fileScheme,
		Path:     filepath.ToSlash(absolute),
		RawQuery: query.Encode(),
	}
	d.pool, err = sql.Open(sqliteDriver, uri.String())
	if err != nil {
		return err
	}
	d.pool.SetMaxOpenConns(maxConnections)
	if err = d.pool.PingContext(ctx); err != nil {
		return err
	}
	_, err = d.pool.ExecContext(ctx, journalStatement)
	return err
}

// SQL exposes operations, while this Bean retains ownership of Close.
func (d *Database) SQL() *sql.DB {
	return d.pool
}

// Destroy also handles initialization failures and repeated owner cleanup.
func (d *Database) Destroy(context.Context) error {
	d.closeOnce.Do(func() {
		if d.pool != nil {
			d.closeErr = d.pool.Close()
		}
	})
	return d.closeErr
}

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
	"database/sql"
	"errors"
	"time"

	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

const (
	columns     = "id,customer,coffee,quantity,unit_price,total,status,version,created_at,updated_at"
	selectOrder = "SELECT " + columns + " FROM orders WHERE id=?"
)

// SQLite implements atomic version/state checks behind the model contract.
type SQLite struct {
	database *infrastructure.Database
}

var _ model.Repository = (*SQLite)(nil)

// NewSQLite requires schema readiness without storing the prerequisite object.
// +ioc:bean
// +ioc:expose=model.Repository
// +ioc:depends-on=databaseMigration
func NewSQLite(database *infrastructure.Database) *SQLite {
	return &SQLite{
		database: database,
	}
}

type rowScanner interface {
	Scan(...any) error
}

func scan(row rowScanner) (model.Order, error) {
	var result model.Order
	var created, updated int64
	err := row.Scan(
		&result.ID, &result.Customer, &result.Coffee, &result.Quantity,
		&result.UnitPrice, &result.Total, &result.Status, &result.Version,
		&created, &updated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Order{}, model.ErrNotFound
	}
	result.CreatedAt = time.UnixMilli(created).UTC()
	result.UpdatedAt = time.UnixMilli(updated).UTC()
	return result, err
}

type transactionFunc func(*sql.Tx) (model.Order, error)

func (r *SQLite) transact(ctx context.Context, operation transactionFunc) (model.Order, error) {
	tx, err := r.database.SQL().BeginTx(ctx, nil)
	if err != nil {
		return model.Order{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	result, err := operation(tx)
	if err != nil {
		return model.Order{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.Order{}, err
	}
	return result, nil
}

func current(ctx context.Context, tx *sql.Tx, mutation model.Mutation) (model.Order, error) {
	order, err := scan(tx.QueryRowContext(ctx, selectOrder, mutation.ID))
	if err != nil {
		return model.Order{}, err
	}
	if order.Version != mutation.Version {
		return model.Order{}, model.ErrConflict
	}
	return order, nil
}

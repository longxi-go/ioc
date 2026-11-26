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

	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

const (
	filterOrders = " FROM orders WHERE (?='' OR status=?)"
	countOrders  = "SELECT count(*)" + filterOrders
	listOrders   = "SELECT " + columns + filterOrders + " ORDER BY id DESC LIMIT ? OFFSET ?"
)

func (r *SQLite) Get(ctx context.Context, id int64) (model.Order, error) {
	if id < 1 {
		return model.Order{}, model.ErrInvalid
	}
	return scan(r.database.SQL().QueryRowContext(ctx, selectOrder, id))
}

func (r *SQLite) List(ctx context.Context, query model.Query) (model.Page, error) {
	if err := query.Validate(); err != nil {
		return model.Page{}, err
	}
	tx, err := r.database.SQL().BeginTx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		return model.Page{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	result := model.Page{
		Orders: []model.Order{},
		Limit:  query.Limit,
		Offset: query.Offset,
	}
	if err = tx.QueryRowContext(ctx, countOrders, query.Status, query.Status).Scan(&result.Total); err != nil {
		return model.Page{}, err
	}
	rows, err := tx.QueryContext(ctx, listOrders, query.Status, query.Status, query.Limit, query.Offset)
	if err != nil {
		return model.Page{}, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		order, err := scan(rows)
		if err != nil {
			return model.Page{}, err
		}
		result.Orders = append(result.Orders, order)
	}
	if err = rows.Err(); err != nil {
		return model.Page{}, err
	}
	if err = rows.Close(); err != nil {
		return model.Page{}, err
	}
	return result, tx.Commit()
}

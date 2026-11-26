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
	"strings"
	"time"

	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

const (
	insertOrder = "INSERT INTO orders(customer,coffee,quantity,unit_price,total,status,version,created_at,updated_at) " +
		"VALUES(?,?,?,?,?,?,1,?,?) RETURNING " + columns
	updateOrder = "UPDATE orders SET customer=?,coffee=?,quantity=?,unit_price=?,total=?,version=version+1,updated_at=? " +
		"WHERE id=? AND version=? AND status=? RETURNING " + columns
	transitionOrder = "UPDATE orders SET status=?,version=version+1,updated_at=? WHERE id=? AND version=? AND status=? RETURNING " + columns
	deleteOrder     = "DELETE FROM orders WHERE id=? AND version=? AND status=?"
)

func (r *SQLite) Create(ctx context.Context, details model.Details, now time.Time) (model.Order, error) {
	coffee, err := details.Validate()
	if err != nil {
		return model.Order{}, err
	}
	return r.transact(ctx, func(tx *sql.Tx) (model.Order, error) {
		return scan(tx.QueryRowContext(ctx, insertOrder,
			strings.TrimSpace(details.Customer), coffee.Code, details.Quantity,
			coffee.Price, coffee.Price*details.Quantity, model.Placed,
			now.UnixMilli(), now.UnixMilli(),
		))
	})
}

func (r *SQLite) Update(ctx context.Context, edit model.Edit, now time.Time) (model.Order, error) {
	if err := edit.Mutation.Validate(); err != nil {
		return model.Order{}, err
	}
	coffee, err := edit.Details.Validate()
	if err != nil {
		return model.Order{}, err
	}
	return r.transact(ctx, func(tx *sql.Tx) (model.Order, error) {
		order, err := current(ctx, tx, edit.Mutation)
		if err != nil {
			return model.Order{}, err
		}
		if order.Status != model.Placed {
			return model.Order{}, model.ErrConflict
		}
		return scan(tx.QueryRowContext(ctx, updateOrder,
			strings.TrimSpace(edit.Customer), coffee.Code, edit.Quantity,
			coffee.Price, coffee.Price*edit.Quantity, now.UnixMilli(),
			edit.ID, edit.Version, model.Placed,
		))
	})
}

func (r *SQLite) Transition(ctx context.Context, change model.Change, now time.Time) (model.Order, error) {
	if err := change.Validate(); err != nil {
		return model.Order{}, err
	}
	if !change.To.Valid() {
		return model.Order{}, model.ErrInvalid
	}
	return r.transact(ctx, func(tx *sql.Tx) (model.Order, error) {
		order, err := current(ctx, tx, change.Mutation)
		if err != nil {
			return model.Order{}, err
		}
		if !order.Status.CanMoveTo(change.To) {
			return model.Order{}, model.ErrConflict
		}
		return scan(tx.QueryRowContext(ctx, transitionOrder, change.To, now.UnixMilli(), change.ID, change.Version, order.Status))
	})
}

func (r *SQLite) Delete(ctx context.Context, mutation model.Mutation) (model.Order, error) {
	if err := mutation.Validate(); err != nil {
		return model.Order{}, err
	}
	return r.transact(ctx, func(tx *sql.Tx) (model.Order, error) {
		order, err := current(ctx, tx, mutation)
		if err != nil {
			return model.Order{}, err
		}
		if order.Status != model.Cancelled {
			return model.Order{}, model.ErrConflict
		}
		if _, err = tx.ExecContext(ctx, deleteOrder, mutation.ID, mutation.Version, model.Cancelled); err != nil {
			return model.Order{}, err
		}
		return order, nil
	})
}

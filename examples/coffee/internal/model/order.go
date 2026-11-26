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

package model

import "time"

// Order is a detached persisted snapshot, safe to return across layers.
type Order struct {
	ID        int64     `json:"id"`
	Customer  string    `json:"customer"`
	Coffee    string    `json:"coffee"`
	Quantity  int64     `json:"quantity"`
	UnitPrice int64     `json:"unitPrice"`
	Total     int64     `json:"total"`
	Status    Status    `json:"status"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Mutation identifies an order and the version observed by the caller.
type Mutation struct {
	ID      int64
	Version int64 `json:"version"`
}

func (m Mutation) Validate() error {
	if m.ID < 1 || m.Version < 1 {
		return ErrInvalid
	}
	return nil
}

// Edit carries editable details; Repository derives prices and preserves creation time.
type Edit struct {
	Mutation
	Details
}

// Change requests one explicit workflow transition.
type Change struct {
	Mutation
	To Status
}

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Query bounds work before accessing the database.
type Query struct {
	Status Status
	Limit  int
	Offset int
}

func (q Query) Validate() error {
	if q.Limit < 1 || q.Limit > MaxLimit || q.Offset < 0 || (q.Status != "" && !q.Status.Valid()) {
		return ErrInvalid
	}
	return nil
}

// Page contains a coherent count and ordered result from one database snapshot.
type Page struct {
	Orders []Order `json:"orders"`
	Total  int64   `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// Operation stores immutable correlation metadata without retaining a request Context.
type Operation struct {
	ID string
}

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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

func ExampleNewSQLite() {
	directory, err := os.MkdirTemp("", "coffee-repository-")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			panic(err)
		}
	}()
	ctx := context.Background()
	settings := &configuration.Settings{
		Database: configuration.Database{
			Path: filepath.Join(directory, "coffee.db"),
		},
	}
	database := infrastructure.NewDatabase(settings)
	defer func() {
		if err := database.Destroy(ctx); err != nil {
			panic(err)
		}
	}()
	if err := database.Init(ctx); err != nil {
		panic(err)
	}
	if err := infrastructure.NewMigration(database).Init(ctx); err != nil {
		panic(err)
	}
	orders := NewSQLite(database)
	order, err := orders.Create(ctx, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 2,
	}, time.Now())
	if err != nil {
		panic(err)
	}
	saved, err := orders.Get(ctx, order.ID)
	fmt.Println(saved.Customer, saved.Total, saved.Version, err)
	// Output: Ada 700 1 <nil>
}

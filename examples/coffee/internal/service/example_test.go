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

package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/longxi-go/gosr/pkg/event"
	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
	"github.com/longxi-go/ioc/examples/coffee/internal/repository"
)

type examplePublisher struct{}

func (examplePublisher) PublishEvent(_ context.Context, fact event.Event) error {
	if changed, ok := fact.(model.OrderChanged); ok {
		fmt.Println(changed.Action, changed.CorrelationID)
	}
	return nil
}

func ExampleService_Create() {
	directory, err := os.MkdirTemp("", "coffee-service-")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			panic(err)
		}
	}()
	ctx := context.Background()
	database := infrastructure.NewDatabase(&configuration.Settings{
		Database: configuration.Database{
			Path: filepath.Join(directory, "coffee.db"),
		},
	})
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
	operation := func(context.Context) (*model.Operation, error) {
		return &model.Operation{
			ID: "request-42",
		}, nil
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	orders := New(repository.NewSQLite(database), examplePublisher{}, operation, logger)
	order, err := orders.Create(ctx, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 2,
	})
	fmt.Println(order.Status, order.Total, err)
	// Output:
	// created request-42
	// placed 700 <nil>
}

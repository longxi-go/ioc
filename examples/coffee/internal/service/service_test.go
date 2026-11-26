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
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/longxi-go/gosr/pkg/event"
	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
	"github.com/longxi-go/ioc/examples/coffee/internal/repository"
)

type failingPublisher struct {
	calls int
}

func (p *failingPublisher) PublishEvent(context.Context, event.Event) error {
	p.calls++
	return errors.New("observer unavailable")
}

func TestObserverFailurePreservesCommittedOrder(t *testing.T) {
	ctx := context.Background()
	settings := &configuration.Settings{
		Database: configuration.Database{
			Path: filepath.Join(t.TempDir(), "service.db"),
		},
	}
	db := infrastructure.NewDatabase(settings)
	if err := db.Init(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Destroy(ctx); err != nil {
			t.Error(err)
		}
	})
	if err := infrastructure.NewMigration(db).Init(ctx); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewSQLite(db)
	publisher := &failingPublisher{}
	operation := func(context.Context) (*model.Operation, error) {
		return &model.Operation{
			ID: "test-1",
		}, nil
	}
	service := New(repo, publisher, operation, slog.New(slog.NewTextHandler(io.Discard, nil)))
	order, err := service.Create(ctx, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 1,
	})
	if err != nil || order.ID == 0 || publisher.calls != 1 {
		t.Fatalf("%+v %v calls=%d", order, err, publisher.calls)
	}
	saved, err := service.Get(ctx, order.ID)
	if err != nil || saved.ID != order.ID {
		t.Fatal(saved, err)
	}
	if _, err = service.Create(ctx, model.Details{}); !errors.Is(err, model.ErrInvalid) || publisher.calls != 1 {
		t.Fatal("published failed write", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = service.Create(cancelled, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 1,
	}); !errors.Is(err, context.Canceled) || publisher.calls != 1 {
		t.Fatal(err)
	}
}

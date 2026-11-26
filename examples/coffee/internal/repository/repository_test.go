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
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

func openTest(t *testing.T, path string) (*SQLite, *infrastructure.Database) {
	t.Helper()
	settings := &configuration.Settings{}
	settings.Database.Path = path
	db := infrastructure.NewDatabase(settings)
	if err := db.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Destroy(context.Background()); err != nil {
			t.Error(err)
		}
	})
	if err := infrastructure.NewMigration(db).Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	return NewSQLite(db), db
}

func TestPersistedCRUDAndReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "coffee.db")
	repo, db := openTest(t, path)
	order, err := repo.Create(ctx, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 2,
	}, time.Now())
	if err != nil || order.Total != 700 || order.Version != 1 {
		t.Fatalf("create: %+v %v", order, err)
	}
	order, err = repo.Update(ctx, model.Edit{
		Mutation: model.Mutation{
			ID:      order.ID,
			Version: order.Version,
		},
		Details: model.Details{
			Customer: "Grace",
			Coffee:   "espresso",
			Quantity: 3,
		},
	}, time.Now())
	if err != nil || order.Total != 750 || order.Version != 2 {
		t.Fatalf("update: %+v %v", order, err)
	}
	if _, err = repo.Delete(ctx, model.Mutation{
		ID:      order.ID,
		Version: order.Version,
	}); !errors.Is(err, model.ErrConflict) {
		t.Fatal("deleted active", err)
	}
	if err = db.Destroy(ctx); err != nil {
		t.Fatal(err)
	}
	repo, _ = openTest(t, path)
	saved, err := repo.Get(ctx, order.ID)
	if err != nil || saved.Customer != "Grace" {
		t.Fatalf("reopen: %+v %v", saved, err)
	}
	page, err := repo.List(ctx, model.Query{
		Limit: 20,
	})
	if err != nil || page.Total != 1 || len(page.Orders) != 1 {
		t.Fatalf("list: %+v %v", page, err)
	}
	if _, err = repo.Update(ctx, model.Edit{
		Mutation: model.Mutation{
			ID:      order.ID,
			Version: 1,
		},
		Details: model.Details{
			Customer: "Lost",
			Coffee:   "latte",
			Quantity: 1,
		},
	}, time.Now()); !errors.Is(err, model.ErrConflict) {
		t.Fatal("stale", err)
	}
	cancelled, err := repo.Transition(ctx, model.Change{
		Mutation: model.Mutation{
			ID:      order.ID,
			Version: order.Version,
		},
		To: model.Cancelled,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Delete(ctx, model.Mutation{
		ID:      order.ID,
		Version: cancelled.Version,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Get(ctx, order.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestConcurrentCollectionHasOneWinner(t *testing.T) {
	ctx := context.Background()
	repo, _ := openTest(t, filepath.Join(t.TempDir(), "coffee.db"))
	order, err := repo.Create(ctx, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 1,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []model.Status{model.Preparing, model.Ready} {
		order, err = repo.Transition(ctx, model.Change{
			Mutation: model.Mutation{
				ID:      order.ID,
				Version: order.Version,
			},
			To: status,
		}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	const workers = 8
	results := make(chan error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			<-start
			_, err := repo.Transition(ctx, model.Change{
				Mutation: model.Mutation{
					ID:      order.ID,
					Version: order.Version,
				},
				To: model.Collected,
			}, time.Now())
			results <- err
		})
	}
	close(start)
	wg.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, model.ErrConflict) {
			t.Fatalf("unexpected: %v", err)
		}
	}
	if wins != 1 {
		t.Fatalf("winners %d", wins)
	}
	if _, err = repo.Update(ctx, model.Edit{
		Mutation: model.Mutation{
			ID:      order.ID,
			Version: order.Version + 1,
		},
		Details: model.Details{
			Customer: "Edit",
			Coffee:   "latte",
			Quantity: 1,
		},
	}, time.Now()); !errors.Is(err, model.ErrConflict) {
		t.Fatal(err)
	}
}

func TestInvalidAndCancelledOperations(t *testing.T) {
	ctx := context.Background()
	repo, _ := openTest(t, filepath.Join(t.TempDir(), "coffee.db"))
	if _, err := repo.Create(ctx, model.Details{}, time.Now()); !errors.Is(err, model.ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, 999); !errors.Is(err, model.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := repo.List(ctx, model.Query{
		Limit: 101,
	}); !errors.Is(err, model.ErrInvalid) {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.Create(cancelled, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 1,
	}, time.Now()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestSQLFailureRollsBackAndConnectionsKeepSettings(t *testing.T) {
	ctx := context.Background()
	repo, db := openTest(t, filepath.Join(t.TempDir(), "rollback.db"))
	const trigger = `CREATE TRIGGER reject_order BEFORE INSERT ON orders WHEN NEW.customer='Rejected' BEGIN SELECT RAISE(ABORT,'rejected'); END;`
	if _, err := db.SQL().ExecContext(ctx, trigger); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, model.Details{
		Customer: "Rejected",
		Coffee:   "latte",
		Quantity: 1,
	}, time.Now()); err == nil {
		t.Fatal("SQL failure not returned")
	}
	page, err := repo.List(ctx, model.Query{
		Limit: 20,
	})
	if err != nil || page.Total != 0 {
		t.Fatal(page, err)
	}
	if _, err := repo.Create(ctx, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 1,
	}, time.Now()); err != nil {
		t.Fatal(err)
	}

	for range 4 {
		connection, err := db.SQL().Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := connection.Close(); err != nil {
				t.Error(err)
			}
		})
		var foreignKeys, busy int
		if err = connection.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			t.Fatal(err)
		}
		if err = connection.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
			t.Fatal(err)
		}
		if foreignKeys != 1 || busy != 5000 {
			t.Fatalf("per-connection settings: %d %d", foreignKeys, busy)
		}
	}
}

func TestListFilterOffsetAndUnknownMutations(t *testing.T) {
	ctx := context.Background()
	repo, _ := openTest(t, filepath.Join(t.TempDir(), "list.db"))
	for range 3 {
		if _, err := repo.Create(ctx, model.Details{
			Customer: "Ada",
			Coffee:   "latte",
			Quantity: 1,
		}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repo.List(ctx, model.Query{
		Status: model.Placed,
		Limit:  1,
		Offset: 1,
	})
	if err != nil || page.Total != 3 || len(page.Orders) != 1 || page.Orders[0].ID != 2 {
		t.Fatal(page, err)
	}
	empty, err := repo.List(ctx, model.Query{
		Status: model.Collected,
		Limit:  20,
	})
	if err != nil || empty.Total != 0 || len(empty.Orders) != 0 {
		t.Fatal(empty, err)
	}
	if _, err := repo.Get(ctx, 0); !errors.Is(err, model.ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := repo.Delete(ctx, model.Mutation{
		ID:      999,
		Version: 1,
	}); !errors.Is(err, model.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := repo.Transition(ctx, model.Change{
		Mutation: model.Mutation{
			ID:      1,
			Version: 1,
		},
		To: model.Status("invalid"),
	}, time.Now()); !errors.Is(err, model.ErrInvalid) {
		t.Fatal(err)
	}
}

func TestCancelAndCollectCompeteForOneVersion(t *testing.T) {
	ctx := context.Background()
	repo, _ := openTest(t, filepath.Join(t.TempDir(), "compete.db"))
	order, err := repo.Create(ctx, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 1,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []model.Status{model.Preparing, model.Ready} {
		order, err = repo.Transition(ctx, model.Change{
			Mutation: model.Mutation{
				ID:      order.ID,
				Version: order.Version,
			},
			To: status,
		}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	results := make(chan error, 2)
	start := make(chan struct{})
	var group sync.WaitGroup
	for _, target := range []model.Status{model.Cancelled, model.Collected} {
		group.Go(func() {
			<-start
			_, err := repo.Transition(ctx, model.Change{
				Mutation: model.Mutation{
					ID:      order.ID,
					Version: order.Version,
				},
				To: target,
			}, time.Now())
			results <- err
		})
	}
	close(start)
	group.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, model.ErrConflict) {
			t.Fatal(err)
		}
	}
	saved, err := repo.Get(ctx, order.ID)
	if err != nil || wins != 1 || saved.Version != order.Version+1 {
		t.Fatal(saved, wins, err)
	}
}

func TestBusyDatabaseIsAnOperationalError(t *testing.T) {
	ctx := context.Background()
	repo, db := openTest(t, filepath.Join(t.TempDir(), "busy.db"))
	locked, err := db.SQL().Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := locked.Close(); err != nil {
			t.Error(err)
		}
	}()
	if _, err = locked.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := locked.ExecContext(ctx, "ROLLBACK"); err != nil {
			t.Error(err)
		}
	}()
	other, err := db.SQL().Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = other.ExecContext(ctx, "PRAGMA busy_timeout=1"); err != nil {
		t.Fatal(err)
	}
	if err = other.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = repo.Create(ctx, model.Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 1,
	}, time.Now())
	if err == nil || errors.Is(err, model.ErrConflict) || errors.Is(err, model.ErrInvalid) || errors.Is(err, model.ErrNotFound) {
		t.Fatal("busy was misclassified", err)
	}
}

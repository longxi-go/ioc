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
	"log/slog"
	"time"

	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

const observerFailureMessage = "order committed; observer delivery failed"

type mutationFunc func() (model.Order, error)

// Service owns use cases and emits facts after successful persistence.
type Service struct {
	repository model.Repository
	publisher  event.Publisher
	operation  gs.Provider[*model.Operation]
	logger     *slog.Logger
}

// New uses ordinary constructor injection; business methods never locate Beans.
// +ioc:bean:name=orderService,aliases={orders}
func New(
	repository model.Repository,
	publisher event.Publisher,
	operation gs.Provider[*model.Operation],
	logger *slog.Logger,
) *Service {
	return &Service{
		repository: repository,
		publisher:  publisher,
		operation:  operation,
		logger:     logger,
	}
}

func (s *Service) commit(ctx context.Context, action model.Action, persist mutationFunc) (model.Order, error) {
	if err := ctx.Err(); err != nil {
		return model.Order{}, err
	}
	operation, err := s.operation(ctx)
	if err != nil {
		return model.Order{}, err
	}
	order, err := persist()
	if err != nil {
		return model.Order{}, err
	}
	fact := model.OrderChanged{
		Action:        action,
		Order:         order,
		CorrelationID: operation.ID,
	}
	if err = s.publisher.PublishEvent(ctx, fact); err != nil {
		s.logger.Error(observerFailureMessage, "order_id", order.ID, "correlation_id", operation.ID, "error", err)
	}
	return order, nil
}

func (s *Service) Create(ctx context.Context, details model.Details) (model.Order, error) {
	persist := func() (model.Order, error) {
		return s.repository.Create(ctx, details, time.Now())
	}
	return s.commit(ctx, model.Created, persist)
}

func (s *Service) Get(ctx context.Context, id int64) (model.Order, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, query model.Query) (model.Page, error) {
	return s.repository.List(ctx, query)
}

func (s *Service) Update(ctx context.Context, edit model.Edit) (model.Order, error) {
	persist := func() (model.Order, error) {
		return s.repository.Update(ctx, edit, time.Now())
	}
	return s.commit(ctx, model.Updated, persist)
}

func (s *Service) Transition(ctx context.Context, change model.Change) (model.Order, error) {
	persist := func() (model.Order, error) {
		return s.repository.Transition(ctx, change, time.Now())
	}
	return s.commit(ctx, model.Transitioned, persist)
}

func (s *Service) Delete(ctx context.Context, mutation model.Mutation) (model.Order, error) {
	persist := func() (model.Order, error) {
		return s.repository.Delete(ctx, mutation)
	}
	return s.commit(ctx, model.Deleted, persist)
}

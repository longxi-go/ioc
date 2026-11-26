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

package observation

import (
	"context"
	"log/slog"

	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

const (
	auditMessage = "order_changed"
	auditOrder   = 100
)

// AuditSink is replaceable independently of event subscription and business code.
type AuditSink interface {
	Record(context.Context, model.OrderChanged) error
}

// ConciseSink is selected only when no eligible custom sink exists.
type ConciseSink struct {
	logger *slog.Logger
}

// NewConciseSink supplies audit logging when no other eligible AuditSink exists.
// +ioc:bean
// +ioc:conditional:on-missing-bean:type=AuditSink
// +ioc:param:parameter=logger,qualifiers={audit}
func NewConciseSink(logger *slog.Logger) AuditSink {
	return &ConciseSink{
		logger: logger,
	}
}

func (s *ConciseSink) Record(ctx context.Context, fact model.OrderChanged) error {
	s.logger.InfoContext(ctx, auditMessage, "action", fact.Action, "order_id", fact.Order.ID, "correlation_id", fact.CorrelationID)
	return nil
}

// VerboseSink is a development-only alternative to the fallback sink.
type VerboseSink struct {
	logger *slog.Logger
}

// NewVerboseSink records complete order snapshots when the dev Profile is active.
// +ioc:bean
// +ioc:profile={dev}
// +ioc:param:parameter=logger,name=auditLogger
func NewVerboseSink(logger *slog.Logger) AuditSink {
	return &VerboseSink{
		logger: logger,
	}
}

func (s *VerboseSink) Record(ctx context.Context, fact model.OrderChanged) error {
	s.logger.InfoContext(ctx, auditMessage, "action", fact.Action, "order", fact.Order, "correlation_id", fact.CorrelationID)
	return nil
}

// AuditListener has a singleton sink, while each callback receives its own Context.
// +ioc:component
// +ioc:listener
type AuditListener struct {
	// +ioc:inject
	sink AuditSink
}

// Order places audit delivery before the receipt method listener.
func (*AuditListener) Order() int {
	return auditOrder
}

func (l *AuditListener) OnEvent(ctx context.Context, fact model.OrderChanged) error {
	return l.sink.Record(ctx, fact)
}

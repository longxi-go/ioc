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

const pickupSlipMessage = "pickup_slip"

// ReceiptPrinter separates slip delivery from order persistence and event subscription.
type ReceiptPrinter interface {
	Print(context.Context, model.Order) error
}

// ConsolePrinter writes order slips to the application Logger.
type ConsolePrinter struct {
	logger *slog.Logger
}

// NewConsolePrinter is eligible only when the receipt feature is enabled.
// +ioc:bean
// +ioc:conditional:on-property:prefix=cafe.features.receipt,name=enabled,havingValue=true
func NewConsolePrinter(logger *slog.Logger) ReceiptPrinter {
	return &ConsolePrinter{
		logger: logger,
	}
}

func (p *ConsolePrinter) Print(ctx context.Context, order model.Order) error {
	p.logger.InfoContext(ctx, pickupSlipMessage, "order_id", order.ID, "coffee", order.Coffee, "quantity", order.Quantity)
	return nil
}

// ReceiptListener is eligible only when a ReceiptPrinter can satisfy its dependency.
// +ioc:component
// +ioc:conditional:on-bean:type=ReceiptPrinter
type ReceiptListener struct {
	// +ioc:inject
	printer ReceiptPrinter
}

// onOrder prints creation events after the audit listener (order 100).
// +ioc:listen
// +ioc:order=200
func (l *ReceiptListener) onOrder(ctx context.Context, fact model.OrderChanged) error {
	if fact.Action != model.Created {
		return nil
	}
	return l.printer.Print(ctx, fact.Order)
}

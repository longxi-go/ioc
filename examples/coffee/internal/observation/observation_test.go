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
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

func TestAuditAndReceiptFacts(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	sink := NewConciseSink(logger)
	event := model.OrderChanged{
		Action: model.Created,
		Order: model.Order{
			ID: 42,
		},
		CorrelationID: "r1",
	}
	if err := sink.Record(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	listener := &ReceiptListener{
		printer: NewConsolePrinter(logger),
	}
	if err := listener.onOrder(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "pickup_slip") || !strings.Contains(output.String(), "order_id=42") {
		t.Fatal(output.String())
	}
	output.Reset()
	event.Action = model.Updated
	if err := listener.onOrder(context.Background(), event); err != nil || output.Len() != 0 {
		t.Fatal(output.String(), err)
	}
}

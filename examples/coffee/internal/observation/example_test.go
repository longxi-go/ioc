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
	"fmt"
	"log/slog"
	"strings"

	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

func ExampleNewConciseSink() {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	audit := NewConciseSink(logger)
	err := audit.Record(context.Background(), model.OrderChanged{
		Action: model.Created,
		Order: model.Order{
			ID: 42,
		},
		CorrelationID: "request-42",
	})
	fmt.Println(err, strings.Contains(output.String(), "order_id=42"))
	// Output: <nil> true
}

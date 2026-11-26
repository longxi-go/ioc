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

package eventdispatch_test

import (
	"context"
	"fmt"

	"github.com/longxi-go/gosr/pkg/event"
	"github.com/longxi-go/ioc/internal/eventdispatch"
)

func Example() {
	// A small resolver supplies existing objects; only the container owns their
	// actual lifetime. Generated adapters have the same direct-call shape.
	dispatcher, err := eventdispatch.New([]eventdispatch.Entry{
		entry("audit", "audit.OnEvent", matchesNote, func(context.Context, any, event.Event) error {
			fmt.Println("received note")
			return nil
		}),
	})
	if err != nil {
		panic(err)
	}
	if err := dispatcher.Deliver(context.Background(), note{}, access(map[string]any{
		"audit": struct{}{},
	})); err != nil {
		panic(err)
	}
	// Output: received note
}

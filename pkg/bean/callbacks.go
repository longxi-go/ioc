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

package bean

import (
	"context"

	"github.com/longxi-go/gosr/pkg/event"
)

// AcceptFunc verifies a generated output contract without runtime discovery.
type AcceptFunc = func(any) bool

// DecoratorFunc wraps an owned value before initialization; the decorator must
// retain or assume cleanup responsibility as documented by Decorator.
type DecoratorFunc = func(context.Context, any) (any, error)

// EventMatchFunc is a pure generated event type check and must not invoke user code.
type EventMatchFunc = func(event.Event) bool

// EventInvokeFunc dispatches directly to a statically verified receiver method.
type EventInvokeFunc = func(context.Context, any, event.Event) error

// ReleaseFunc ends a caller-owned acquisition. The issuer retains responsibility
// for making repeated release safe and for completing resource cleanup.
type ReleaseFunc = func(context.Context) error

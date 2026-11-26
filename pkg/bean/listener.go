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

import gs "github.com/longxi-go/gosr/pkg/ioc"

const (
	// EventPublisherID belongs to per-context infrastructure, never a generated
	// catalog contribution. Application plans omit it from contribution IDs.
	EventPublisherID = "ioc.infrastructure/eventPublisher"
	// EventPublisherName permits explicit selection of the local default while
	// protecting its identity against user names and aliases.
	EventPublisherName = "iocEventPublisher"
	// EventPublisherContract is the generated canonical capability identity.
	EventPublisherContract gs.Contract = "github.com/longxi-go/gosr/pkg/event.Publisher"
)

// Listener is a static method adapter owned by its containing Registration.
// Match must be a pure generated type assertion; Invoke performs a direct call.
// Neither runs during catalog contribution. ID identifies a declaration rather
// than the whole object; Event is diagnostic metadata, not a routing topic.
// Order overrides the owner's sampled numeric order while retaining its priority
// group; nil uses Ordered or LowestPrecedence. Mutable metadata is copied.
type Listener struct {
	ID, Event string
	Source    gs.Source
	Order     *int
	Match     EventMatchFunc
	Invoke    EventInvokeFunc
}

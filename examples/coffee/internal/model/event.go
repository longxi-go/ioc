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

package model

import "github.com/longxi-go/gosr/pkg/event"

// Action distinguishes persisted facts from requested commands.
type Action string

const (
	Created      Action = "created"
	Updated      Action = "updated"
	Transitioned Action = "transitioned"
	Deleted      Action = "deleted"
)

// OrderChanged is published only after its database transaction commits.
type OrderChanged struct {
	event.BaseEvent
	Action        Action
	Order         Order
	CorrelationID string
}

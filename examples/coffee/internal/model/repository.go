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

import (
	"context"
	"time"
)

// Repository owns atomic version/state checks as well as persistence.
type Repository interface {
	Create(context.Context, Details, time.Time) (Order, error)
	Get(context.Context, int64) (Order, error)
	List(context.Context, Query) (Page, error)
	Update(context.Context, Edit, time.Time) (Order, error)
	Transition(context.Context, Change, time.Time) (Order, error)
	Delete(context.Context, Mutation) (Order, error)
}

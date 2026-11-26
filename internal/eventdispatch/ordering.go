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

package eventdispatch

import (
	"cmp"
	"slices"

	"github.com/longxi-go/gosr/pkg/ordered"
)

type orderKey struct {
	group, order int
}

func sample(value any) (orderKey, error) {
	key := orderKey{
		group: 1,
		order: ordered.LowestPrecedence,
	}
	if _, ok := value.(ordered.PriorityOrdered); ok {
		key.group = 0
	}
	err := protect(func() error {
		if ordered, ok := value.(ordered.Ordered); ok {
			key.order = ordered.Order()
		}
		return nil
	})
	return key, err
}

func sortInvocations(calls []invocation) {
	slices.SortFunc(calls, func(a, b invocation) int {
		return cmp.Or(
			cmp.Compare(a.key.group, b.key.group),
			cmp.Compare(a.key.order, b.key.order),
			cmp.Compare(a.entry.Listener.ID, b.entry.Listener.ID),
		)
	})
}

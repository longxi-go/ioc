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
	"errors"
	"testing"
)

func TestDetailsPriceAndBounds(t *testing.T) {
	d := Details{
		Customer: "Ada",
		Coffee:   "latte",
		Quantity: 2,
	}
	item, err := d.Validate()
	if err != nil || item.Price*d.Quantity != 700 {
		t.Fatalf("price: %v %v", item, err)
	}
	for _, d := range []Details{
		{
			Customer: "",
			Coffee:   "latte",
			Quantity: 1,
		},
		{
			Customer: "Ada",
			Coffee:   "missing",
			Quantity: 1,
		},
		{
			Customer: "Ada",
			Coffee:   "latte",
			Quantity: 0,
		},
		{
			Customer: "Ada",
			Coffee:   "latte",
			Quantity: 101,
		},
	} {
		if _, err := d.Validate(); !errors.Is(err, ErrInvalid) {
			t.Fatalf("accepted %#v: %v", d, err)
		}
	}
}

func TestTransitions(t *testing.T) {
	for _, current := range []Status{Placed, Preparing, Ready, Collected, Cancelled} {
		for _, target := range []Status{Placed, Preparing, Ready, Collected, Cancelled, Status("unknown")} {
			want := current == Placed && (target == Preparing || target == Cancelled) ||
				current == Preparing && (target == Ready || target == Cancelled) ||
				current == Ready && (target == Collected || target == Cancelled)
			if got := current.CanMoveTo(target); got != want {
				t.Errorf("%s -> %s: %v", current, target, got)
			}
		}
	}
}

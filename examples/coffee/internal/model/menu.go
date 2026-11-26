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
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	MaxQuantity       int64 = 100
	MaxCustomerLength int   = 80
)

// Coffee has a server-owned price in minor currency units.
type Coffee struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

// Menu returns detached fixed menu data; customers never choose unit prices.
func Menu() []Coffee {
	return []Coffee{
		{
			Code:  "espresso",
			Name:  "Espresso",
			Price: 250,
		},
		{
			Code:  "americano",
			Name:  "Americano",
			Price: 300,
		},
		{
			Code:  "latte",
			Name:  "Latte",
			Price: 350,
		},
	}
}

// Details contains only the customer-editable portion of an order.
type Details struct {
	Customer string `json:"customer"`
	Coffee   string `json:"coffee"`
	Quantity int64  `json:"quantity"`
}

// Validate checks bounds before calculating integer prices.
func (d Details) Validate() (Coffee, error) {
	if strings.TrimSpace(d.Customer) == "" || utf8.RuneCountInString(d.Customer) > MaxCustomerLength ||
		!utf8.ValidString(d.Customer) || d.Quantity < 1 || d.Quantity > MaxQuantity {
		return Coffee{}, fmt.Errorf("%w: customer or quantity", ErrInvalid)
	}
	for _, coffee := range Menu() {
		if coffee.Code == d.Coffee {
			return coffee, nil
		}
	}
	return Coffee{}, fmt.Errorf("%w: unknown coffee", ErrInvalid)
}

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

package view

import (
	"strings"
	"testing"

	"github.com/longxi-go/ioc/examples/coffee/internal/model"
)

func TestEmbeddedPagesEscapeCustomer(t *testing.T) {
	renderer, err := New()
	if err != nil {
		t.Fatal(err)
	}
	body, err := renderer.Render(Detail, Page{
		Title: "Order",
		Order: model.Order{
			ID:       1,
			Customer: "<script>alert(1)</script>",
			Status:   model.Placed,
		},
		Menu: model.Menu(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "<script>alert(1)</script>") || !strings.Contains(string(body), "&lt;script&gt;") {
		t.Fatal(string(body))
	}
	if _, err = renderer.Render("missing", Page{}); err == nil {
		t.Fatal("unknown template accepted")
	}
}

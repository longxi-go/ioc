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

package integration_test

import (
	"context"
	"fmt"

	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/ioc/pkg/catalog"
	appcontext "github.com/longxi-go/ioc/pkg/context"
)

func Example() {
	c := catalog.New()
	r := registration("message")
	c.Contribute(bean.Batch{
		ID:            "example",
		Schema:        bean.SchemaVersion,
		Registrations: []bean.Registration{r},
	})
	c.Install(bean.Plan{
		ID:            "example",
		DefinitionIDs: []string{"message"},
		Query:         query,
	})
	app, err := appcontext.NewApplicationContext(context.Background(), appcontext.WithCatalog(c))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		panic(err)
	}
	value, err := app.GetBean[*item](ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(value.name, app.State())
	if err := app.Close(ctx); err != nil {
		panic(err)
	}
	// Output: message running
}

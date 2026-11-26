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

package context_test

import (
	"context"
	"fmt"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/ioc/pkg/catalog"
	managed "github.com/longxi-go/ioc/pkg/context"
)

func ExampleApplication() {
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:     "example",
		Schema: bean.SchemaVersion,
		Registrations: []bean.Registration{
			{
				Definition: gs.BeanDefinition{
					ID:        "answer",
					Name:      "answer",
					Contracts: []gs.Contract{"int"},
					Scope:     gs.Singleton,
				},
				Create: func(context.Context, bean.Ports) (any, error) {
					return 42, nil
				},
			},
		},
	})
	a, err := managed.NewApplicationContext(context.Background(), managed.WithCatalog(c))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if err = a.Start(ctx); err != nil {
		panic(err)
	}
	value, err := a.Lookup(ctx, gs.Selector{
		Name: "answer",
	})
	fmt.Println(value, err)
	if err = a.Close(ctx); err != nil {
		panic(err)
	}
	// Output: 42 <nil>
}

// Generated registrations can be inspected before Start; this example uses an
// isolated catalog so it does not depend on packages imported by the caller.
func ExampleApplication_ConditionReport() {
	a, err := managed.NewApplicationContext(context.Background(), managed.WithCatalog(catalog.New()))
	if err != nil {
		panic(err)
	}
	report, err := a.ConditionReport()
	if err != nil {
		panic(err)
	}
	fmt.Println(a.State())
	for _, candidate := range report.Candidates {
		fmt.Println(candidate.Outcome)
	}
	if err := a.Close(context.Background()); err != nil {
		panic(err)
	}
	// Output:
	// created
	// accepted
	// accepted
}

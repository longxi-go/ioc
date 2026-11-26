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
	"testing"

	"github.com/longxi-go/ioc/pkg/catalog"
	app "github.com/longxi-go/ioc/pkg/context"
)

func TestProcessValuesRemainLiteral(t *testing.T) {
	for _, value := range []string{"${unknown}", "${text", "\\${literal}", "${-*-command-variables-*-}"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("IOC_LITERAL_TEST", value)
			t.Setenv("MAKEOVERRIDES", "${-*-command-variables-*-}")
			application, err := app.NewApplicationContext(
				context.Background(),
				app.WithCatalog(catalog.New()),
				app.WithProperties(map[string]string{
					"literal.reference": "${ioc.literal.test}",
				}),
			)
			if err != nil {
				t.Fatal("literal process bootstrap failed", err)
			}
			defer func() {
				if err := application.Close(context.Background()); err != nil {
					t.Error(err)
				}
			}()
			for _, key := range []string{"ioc.literal.test", "literal.reference"} {
				property, found, err := application.Environment().Lookup(key)
				if err != nil || !found {
					t.Fatal("lookup failed", err)
				}
				got, _ := property.Value.Scalar()
				if got != value {
					t.Fatal("process value was changed")
				}
			}
		})
	}
}

func TestProgrammaticValuesStillResolveStrictly(t *testing.T) {
	_, err := app.NewApplicationContext(
		context.Background(),
		app.WithCatalog(catalog.New()),
		app.WithProperties(map[string]string{
			"literal.reference": "${ioc.nonexistent.literal.reference}",
		}),
	)
	if err == nil {
		t.Fatal("unresolved programmatic reference succeeded")
	}
}

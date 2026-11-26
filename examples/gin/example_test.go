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

package main

import (
	"context"
	"fmt"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc"
)

func ExampleHandler_Describe() {
	ctx := context.Background()
	owner, err := ioc.NewApplicationContext(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := owner.Close(ctx); err != nil {
			panic(err)
		}
	}()
	if err := owner.Start(ctx); err != nil {
		panic(err)
	}
	handler, err := owner.GetBean[*Handler](ctx)
	if err != nil {
		panic(err)
	}
	request, scope, err := owner.OpenScope(context.WithValue(ctx, tenantKey{}, demonstrationTenant), gs.Request)
	if err != nil {
		panic(err)
	}
	view, err := handler.Describe(request)
	if err != nil {
		panic(err)
	}
	if err := scope.Close(ctx); err != nil {
		panic(err)
	}
	fmt.Println(view.Tenant, view.FirstID, view.SecondID)
	// Output: alpha 1 1
}

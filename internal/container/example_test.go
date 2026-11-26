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

package container

import (
	"context"
	"fmt"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func ExampleEngine() {
	r := recipe("answer", gs.Singleton, func(context.Context, bean.Ports) (any, error) {
		return 42, nil
	})
	e, err := New([]bean.Registration{r}, Options{})
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if err = e.Start(ctx); err != nil {
		panic(err)
	}
	value, err := e.Lookup(ctx, gs.Selector{
		Name: "answer",
	})
	fmt.Println(value, err)
	if err = e.Close(ctx); err != nil {
		panic(err)
	}
	// Output: 42 <nil>
}

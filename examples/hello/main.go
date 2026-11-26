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
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc"
	"github.com/longxi-go/ioc/examples/hello/contracts"
)

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, out io.Writer) (result error) {
	app, err := ioc.NewApplicationContext(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		result = errors.Join(result, app.Close(closeCtx))
	}()
	if err := app.Start(ctx); err != nil {
		return err
	}
	request, scope, err := app.OpenScope(ctx, gs.Request)
	if err != nil {
		return err
	}
	defer func() {
		result = errors.Join(result, scope.Close(context.WithoutCancel(ctx)))
	}()
	greeter, err := app.GetBean[contracts.Greeter](ctx)
	if err != nil {
		return err
	}
	message, err := greeter.Greet(request)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, message)
	return err
}

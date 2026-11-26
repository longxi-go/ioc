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

	"github.com/longxi-go/ioc"
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
		result = errors.Join(result, app.Close(context.WithoutCancel(ctx)))
	}()
	if err := app.Start(ctx); err != nil {
		return err
	}
	repository, err := app.GetBean[*Repository](ctx)
	if err != nil {
		return err
	}
	for _, entry := range repository.trace.Entries {
		if _, err := fmt.Fprintln(out, entry); err != nil {
			return err
		}
	}
	return nil
}

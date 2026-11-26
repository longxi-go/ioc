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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
)

const demonstrationTenant = "alpha"

func reportError(err error) {
	slog.Error("request processing failed", "error", err)
}

func run(ctx context.Context, output io.Writer) (err error) {
	app, err := New(ctx, reportError)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
		defer cancel()
		err = errors.Join(err, app.Close(cleanup))
	}()
	input := httptest.NewRequestWithContext(ctx, http.MethodGet, requestPath, nil)
	input.Header.Set(tenantHeader, demonstrationTenant)
	response := httptest.NewRecorder()
	app.Router().ServeHTTP(response, input)
	if response.Code != http.StatusOK {
		return errors.New(requestUnavailable)
	}
	_, err = io.Copy(output, response.Body)
	return err
}

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

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
	"net/http"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

func TestShutdownRejectsNewRequestsAndIsRepeatable(t *testing.T) {
	app, _ := startApplication(t)
	if response := request(t.Context(), app.Router(), "/request", "alpha"); response.Code != http.StatusOK {
		t.Fatal(response.Code)
	}
	for range 2 {
		if err := app.Close(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	response := request(t.Context(), app.Router(), "/request", "alpha")
	if response.Code != http.StatusServiceUnavailable || app.context.State() != gs.Closed || app.statistics.destroyed.Load() != 1 {
		t.Fatalf("status=%d state=%s destroyed=%d", response.Code, app.context.State(), app.statistics.destroyed.Load())
	}
}

func TestNilErrorHandlerRejected(t *testing.T) {
	if _, err := New(t.Context(), nil); err == nil {
		t.Fatal("nil error handler accepted")
	}
}

func TestRunDemonstration(t *testing.T) {
	if err := run(t.Context(), io.Discard); err != nil {
		t.Fatal(fmt.Errorf("run: %w", err))
	}
}

func TestCanceledStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	log := &errorLog{}
	if _, err := New(ctx, log.add); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled construction: %v", err)
	}
}

func TestApplicationCloseReleasesOutstandingScope(t *testing.T) {
	app, _ := startApplication(t)
	parent := context.WithValue(t.Context(), tenantKey{}, "alpha")
	request, scope, err := app.context.OpenScope(parent, gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.handler.Describe(request); err != nil {
		t.Fatal(err)
	}
	if err := app.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := scope.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if app.statistics.destroyed.Load() != 1 {
		t.Fatalf("outstanding scope destroyed=%d times", app.statistics.destroyed.Load())
	}
}

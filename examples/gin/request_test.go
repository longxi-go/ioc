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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestGeneratedInjectionAndRequestOwnership(t *testing.T) {
	app, log := startApplication(t)
	previous := int64(0)
	for range 2 {
		response := request(t.Context(), app.Router(), "/request", "alpha")
		var view View
		if err := json.NewDecoder(response.Body).Decode(&view); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusOK || view.FirstID != view.SecondID || view.FirstID <= previous || view.Tenant != "alpha" {
			t.Fatalf("status=%d view=%+v", response.Code, view)
		}
		previous = view.FirstID
	}
	if app.statistics.created.Load() != 2 || app.statistics.destroyed.Load() != 2 || len(log.snapshot()) != 0 {
		t.Fatalf("created=%d destroyed=%d errors=%v", app.statistics.created.Load(), app.statistics.destroyed.Load(), log.snapshot())
	}
}

func TestConcurrentRequestsAreIsolated(t *testing.T) {
	app, _ := startApplication(t)
	const count = 24
	results := make(chan *httptest.ResponseRecorder, count)
	var requests sync.WaitGroup
	for range count {
		requests.Go(func() {
			results <- request(t.Context(), app.Router(), "/request", "alpha")
		})
	}
	requests.Wait()
	close(results)
	ids := make(map[int64]bool, count)
	for response := range results {
		var view View
		if err := json.NewDecoder(response.Body).Decode(&view); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusOK || ids[view.FirstID] || view.FirstID != view.SecondID {
			t.Fatalf("shared request instance or failed response: %+v status=%d", view, response.Code)
		}
		ids[view.FirstID] = true
	}
	if app.statistics.created.Load() != count || app.statistics.destroyed.Load() != count {
		t.Fatalf("created=%d destroyed=%d", app.statistics.created.Load(), app.statistics.destroyed.Load())
	}
}

func TestConstructorFailureDoesNotPoisonApplication(t *testing.T) {
	app, log := startApplication(t)
	failed := request(t.Context(), app.Router(), "/request", "")
	if failed.Code != http.StatusServiceUnavailable || app.statistics.created.Load() != 0 {
		t.Fatalf("invalid tenant: status=%d created=%d", failed.Code, app.statistics.created.Load())
	}
	if !errors.Is(errors.Join(log.snapshot()...), errMissingTenant) {
		t.Fatalf("constructor error not reported: %v", log.snapshot())
	}
	valid := request(t.Context(), app.Router(), "/request", "alpha")
	if valid.Code != http.StatusOK || app.statistics.created.Load() != 1 || app.statistics.destroyed.Load() != 1 {
		t.Fatalf("later request failed: status=%d created=%d destroyed=%d",
			valid.Code, app.statistics.created.Load(), app.statistics.destroyed.Load(),
		)
	}
}

func TestCanceledRequestIsRejectedBeforeAllocation(t *testing.T) {
	app, log := startApplication(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	response := request(ctx, app.Router(), "/request", "alpha")
	if response.Code != http.StatusServiceUnavailable || app.statistics.created.Load() != 0 {
		t.Fatalf("status=%d created=%d", response.Code, app.statistics.created.Load())
	}
	if !errors.Is(errors.Join(log.snapshot()...), context.Canceled) {
		t.Fatalf("cancellation not reported: %v", log.snapshot())
	}
}

func TestGeneratedRouteDeclarations(t *testing.T) {
	routes := IOCGinRoutes()
	if len(routes) != 1 || routes[0].Method != http.MethodGet || routes[0].Path != requestPath || routes[0].Bean == "" || routes[0].Bind == nil {
		t.Fatalf("generated routes: %+v", routes)
	}
}

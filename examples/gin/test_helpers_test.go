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
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type errorLog struct {
	mu     sync.Mutex
	values []error
}

func (l *errorLog) add(err error) {
	l.mu.Lock()
	l.values = append(l.values, err)
	l.mu.Unlock()
}

func (l *errorLog) snapshot() []error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]error(nil), l.values...)
}

func startApplication(t *testing.T) (*Application, *errorLog) {
	t.Helper()
	log := &errorLog{}
	app, err := New(t.Context(), log.add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return app, log
}

func request(ctx context.Context, router http.Handler, path, tenant string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	input := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	input.Header.Set("X-Tenant", tenant)
	router.ServeHTTP(response, input)
	return response
}

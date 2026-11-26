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

package controller

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/longxi-go/gosr/pkg/event"
	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
	"github.com/longxi-go/ioc/examples/coffee/internal/repository"
	"github.com/longxi-go/ioc/examples/coffee/internal/service"
	"github.com/longxi-go/ioc/examples/coffee/internal/view"
)

type noEvents struct{}

func (noEvents) PublishEvent(context.Context, event.Event) error {
	return nil
}

func routerTest(t *testing.T) *gin.Engine {
	t.Helper()
	ctx := context.Background()
	db := infrastructure.NewDatabase(&configuration.Settings{
		Database: configuration.Database{
			Path: filepath.Join(t.TempDir(), "http.db"),
		},
	})
	if err := db.Init(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Destroy(ctx); err != nil {
			t.Error(err)
		}
	})
	if err := infrastructure.NewMigration(db).Init(ctx); err != nil {
		t.Fatal(err)
	}
	operation := func(context.Context) (*model.Operation, error) {
		return &model.Operation{
			ID: "controller-test",
		}, nil
	}
	orders := service.New(repository.NewSQLite(db), noEvents{}, operation, slog.New(slog.NewTextHandler(io.Discard, nil)))
	renderer, err := view.New()
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	(&API{
		orders: orders,
	}).Register(router)
	(&Pages{
		orders: orders,
		view:   renderer,
	}).Register(router)
	return router
}

func dispatch(router http.Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestAPIAndHTMLCRUD(t *testing.T) {
	router := routerTest(t)
	response := dispatch(router, "POST", "/api/orders", `{"customer":"<Ada>","coffee":"latte","quantity":2}`, "application/json")
	if response.Code != 201 {
		t.Fatal(response.Code, response.Body.String())
	}
	var order model.Order
	if err := json.Unmarshal(response.Body.Bytes(), &order); err != nil {
		t.Fatal(err)
	}
	if order.Total != 700 || order.Version != 1 {
		t.Fatal(order)
	}
	page := dispatch(router, "GET", "/orders/1", "", "")
	if page.Code != 200 || !strings.Contains(page.Body.String(), "&lt;Ada&gt;") || strings.Contains(page.Body.String(), `value="<Ada>"`) {
		t.Fatal(page.Code, page.Body.String())
	}
	update := dispatch(router, "PUT", "/api/orders/1", `{"customer":"Grace","coffee":"espresso","quantity":3,"version":1}`, "application/json")
	if update.Code != 200 {
		t.Fatal(update.Code, update.Body.String())
	}
	stale := dispatch(router, "PUT", "/api/orders/1", `{"customer":"Lost","coffee":"espresso","quantity":1,"version":1}`, "application/json")
	if stale.Code != 409 {
		t.Fatal(stale.Code, stale.Body.String())
	}
	cancel := dispatch(router, "POST", "/orders/1/cancel", "version=2", "application/x-www-form-urlencoded")
	if cancel.Code != 303 {
		t.Fatal(cancel.Code, cancel.Body.String())
	}
	remove := dispatch(router, "DELETE", "/api/orders/1?version=3", "", "")
	if remove.Code != 204 {
		t.Fatal(remove.Code, remove.Body.String())
	}
	if get := dispatch(router, "GET", "/api/orders/1", "", ""); get.Code != 404 {
		t.Fatal(get.Code)
	}
}

func TestInvalidPayloads(t *testing.T) {
	router := routerTest(t)
	for _, body := range []string{
		`{`, `{} {}`,
		`{"customer":"Ada","coffee":"latte","quantity":1,"price":1}`,
		`{"customer":"Ada","coffee":"latte","quantity":1} {}`,
		strings.Repeat("x", 20000),
	} {
		response := dispatch(router, "POST", "/api/orders", body, "application/json")
		if response.Code != 400 {
			t.Fatalf("payload: %d %s", response.Code, response.Body.String())
		}
	}
	if result := dispatch(router, "POST", "/api/orders", `{}`, "text/plain"); result.Code != 415 {
		t.Fatal(result.Code)
	}
	for _, path := range []string{"/api/orders?limit=101", "/api/orders?offset=-1", "/api/orders?status=unknown", "/api/orders/nope"} {
		if result := dispatch(router, "GET", path, "", ""); result.Code != 400 {
			t.Fatal(path, result.Code, result.Body.String())
		}
	}
}

func TestHTMLFormsAndErrorPages(t *testing.T) {
	router := routerTest(t)
	for _, path := range []string{"/", "/orders/new", "/assets/style.css"} {
		if result := dispatch(router, "GET", path, "", ""); result.Code != 200 {
			t.Fatal(path, result.Code, result.Body.String())
		}
	}
	created := dispatch(router, "POST", "/orders", "customer=Ada&coffee=latte&quantity=1", "application/x-www-form-urlencoded")
	if created.Code != 303 || created.Header().Get("Location") != "/orders/1" {
		t.Fatal(created.Code, created.Body.String())
	}
	updated := dispatch(router, "POST", "/orders/1/update", "customer=Grace&coffee=espresso&quantity=2&version=1", "application/x-www-form-urlencoded")
	if updated.Code != 303 {
		t.Fatal(updated.Code, updated.Body.String())
	}
	for _, step := range []struct {
		path, version string
	}{
		{
			"prepare",
			"2",
		},
		{
			"ready",
			"3",
		},
		{
			"cancel",
			"4",
		},
		{
			"delete",
			"5",
		},
	} {
		result := dispatch(router, "POST", "/orders/1/"+step.path, "version="+step.version, "application/x-www-form-urlencoded")
		if result.Code != 303 {
			t.Fatal(step.path, result.Code, result.Body.String())
		}
	}
	for _, test := range []struct {
		path   string
		status int
	}{
		{
			"/orders/1",
			404,
		},
		{
			"/orders/nope",
			400,
		},
		{
			"/?limit=-1",
			400,
		},
	} {
		result := dispatch(router, "GET", test.path, "", "")
		if result.Code != test.status || !strings.Contains(result.Body.String(), "REQUEST NOT COMPLETED") {
			t.Fatal(result.Code, result.Body.String())
		}
	}
	for _, body := range []string{
		"customer=Ada&coffee=latte&quantity=-1",
		"customer=Ada&customer=Eve&coffee=latte&quantity=1",
		"extra=ignored", "customer=%ZZ",
	} {
		result := dispatch(router, "POST", "/orders", body, "application/x-www-form-urlencoded")
		if result.Code != 400 {
			t.Fatal(body, result.Code)
		}
	}
}

func TestSizeLimitsWithOtherwiseValidPayloads(t *testing.T) {
	router := routerTest(t)
	jsonBody := `{"customer":"Ada","coffee":"latte","quantity":1}` + strings.Repeat(" ", 17000)
	formBody := "customer=Ada&coffee=latte&quantity=1" + strings.Repeat("&", 17000)
	for _, test := range []struct {
		path, body, contentType string
	}{
		{
			"/api/orders",
			jsonBody,
			"application/json",
		},
		{
			"/orders",
			formBody,
			"application/x-www-form-urlencoded",
		},
	} {
		response := dispatch(router, "POST", test.path, test.body, test.contentType)
		if response.Code != 400 {
			t.Fatal("size limit bypass", response.Code, response.Body.String())
		}
	}
	response := dispatch(router, "GET", "/api/orders", "", "")
	var page model.Page
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil || page.Total != 0 {
		t.Fatal("oversized input persisted", page, err)
	}
}

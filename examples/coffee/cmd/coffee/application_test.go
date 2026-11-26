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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/examples/coffee/internal/service"

	"github.com/longxi-go/ioc/examples/coffee/internal/application"
	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	"github.com/longxi-go/ioc/examples/coffee/internal/infrastructure"
	"github.com/longxi-go/ioc/examples/coffee/internal/model"
	"github.com/longxi-go/ioc/examples/coffee/internal/observation"
	"github.com/longxi-go/ioc/pkg/bean"
	managed "github.com/longxi-go/ioc/pkg/context"
	nemoenv "github.com/longxi-go/nemo/pkg/environment"
)

func start(t *testing.T, path string, dev, receipts bool) *application.Running {
	t.Helper()
	properties := map[string]string{configuration.DatabaseKey: path, configuration.AddressKey: "127.0.0.1:8082", configuration.ReceiptKey: "false"}
	if receipts {
		properties[configuration.ReceiptKey] = "true"
	}
	options := []managed.Option{managed.WithProperties(properties)}
	if dev {
		options = append(options, managed.WithEnvironmentOptions(nemoenv.WithProfiles("dev")))
	}
	app, err := application.Start(t.Context(), options...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return app
}

func call(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestGeneratedFlowAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coffee.db")
	app := start(t, path, false, false)
	response := call(app.Router(), "POST", "/api/orders", `{"customer":"Ada","coffee":"latte","quantity":2}`)
	if response.Code != 201 {
		t.Fatal(response.Code, response.Body.String())
	}
	for _, step := range []struct {
		action, body string
	}{
		{
			"prepare",
			`{"version":1}`,
		},
		{
			"ready",
			`{"version":2}`,
		},
		{
			"collect",
			`{"version":3}`,
		},
	} {
		result := call(app.Router(), "POST", "/api/orders/1/"+step.action, step.body)
		if result.Code != 200 {
			t.Fatal(result.Code, result.Body.String())
		}
	}
	if result := call(app.Router(), "POST", "/api/orders/1/collect", `{"version":3}`); result.Code != 409 {
		t.Fatal(result.Code)
	}
	db, err := app.Context.GetBean[*infrastructure.Database](t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = db.SQL().PingContext(t.Context()); err == nil {
		t.Fatal("database leaked after close")
	}
	app = start(t, path, false, false)
	saved := call(app.Router(), "GET", "/api/orders/1", "")
	var order model.Order
	if err = json.Unmarshal(saved.Body.Bytes(), &order); err != nil || order.Status != model.Collected || order.Version != 4 {
		t.Fatal(order, err)
	}
	foreign := httptest.NewRequest("POST", "http://localhost/orders", strings.NewReader("customer=Eve&coffee=latte&quantity=1"))
	foreign.Header.Set("Origin", "https://attacker.test")
	foreign.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	denied := httptest.NewRecorder()
	app.Router().ServeHTTP(denied, foreign)
	if denied.Code != 403 {
		t.Fatal("cross-origin mutation", denied.Code)
	}
}

func TestConditionalVariants(t *testing.T) {
	for _, test := range []struct {
		name          string
		dev, receipts bool
	}{
		{
			"default",
			false,
			false,
		},
		{
			"dev",
			true,
			false,
		},
		{
			"receipt",
			false,
			true,
		},
		{
			"dev-receipt",
			true,
			true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := start(t, filepath.Join(t.TempDir(), "coffee.db"), test.dev, test.receipts)
			sink, err := app.Context.GetBean[observation.AuditSink](t.Context())
			if err != nil {
				t.Fatal(err)
			}
			_, verbose := sink.(*observation.VerboseSink)
			if verbose != test.dev {
				t.Fatalf("sink %T", sink)
			}
			printers, err := app.Context.GetBeans[observation.ReceiptPrinter](t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if (len(printers) == 1) != test.receipts {
				t.Fatalf("printers %d", len(printers))
			}
			report, err := app.Context.ConditionReport()
			if err != nil {
				t.Fatal(err)
			}
			checked := 0
			for _, candidate := range report.Candidates {
				if strings.HasSuffix(candidate.Bean, ".ReceiptListener") || strings.HasSuffix(candidate.Bean, ".NewConsolePrinter") {
					checked++
					if (candidate.Outcome == bean.Accepted) != test.receipts {
						t.Fatal(candidate)
					}
				}
			}
			if checked != 2 {
				t.Fatalf("condition coverage %d", checked)
			}
			response := call(app.Router(), "POST", "/api/orders", `{"customer":"Ada","coffee":"latte","quantity":1}`)
			if response.Code != 201 {
				t.Fatal(response.Code, response.Body.String())
			}
		})
	}
}

func TestInvalidStartup(t *testing.T) {
	_, err := application.Start(t.Context(), managed.WithProperties(map[string]string{configuration.DatabaseKey: filepath.Join(t.TempDir(), "missing", "coffee.db")}))
	if err == nil {
		t.Fatal("invalid database path accepted")
	}
}

type traceSink struct {
	next  observation.AuditSink
	trace *[]string
}

func (s *traceSink) Record(ctx context.Context, fact model.OrderChanged) error {
	*s.trace = append(*s.trace, "audit")
	return s.next.Record(ctx, fact)
}

type tracePrinter struct {
	next  observation.ReceiptPrinter
	trace *[]string
}

func (p *tracePrinter) Print(ctx context.Context, order model.Order) error {
	*p.trace = append(*p.trace, "receipt")
	return p.next.Print(ctx, order)
}

func TestGeneratedListenerOrderAndInterfaceDecoration(t *testing.T) {
	var trace []string
	sinkWrap := func(_ context.Context, value any) (any, error) {
		return &traceSink{
			next:  value.(observation.AuditSink),
			trace: &trace,
		}, nil
	}
	printerWrap := func(_ context.Context, value any) (any, error) {
		return &tracePrinter{
			next:  value.(observation.ReceiptPrinter),
			trace: &trace,
		}, nil
	}
	app, err := application.Start(t.Context(), managed.WithProperties(map[string]string{configuration.DatabaseKey: filepath.Join(t.TempDir(), "events.db"), configuration.ReceiptKey: "true"}),
		managed.WithDecorator(bean.Decorator{
			Contract: gs.Contract("github.com/longxi-go/ioc/examples/coffee/internal/observation.AuditSink"),
			Wrap:     sinkWrap,
		}),
		managed.WithDecorator(bean.Decorator{
			Contract: gs.Contract("github.com/longxi-go/ioc/examples/coffee/internal/observation.ReceiptPrinter"),
			Wrap:     printerWrap,
		}))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := app.Close(context.Background()); err != nil {
			t.Error(err)
		}
	}()
	response := call(app.Router(), "POST", "/api/orders", `{"customer":"Ada","coffee":"latte","quantity":1}`)
	if response.Code != 201 || strings.Join(trace, ",") != "audit,receipt" {
		t.Fatalf("%d trace=%v %s", response.Code, trace, response.Body.String())
	}
}

func TestAliasesPrimarySelectionAndClosedAdmission(t *testing.T) {
	app := start(t, filepath.Join(t.TempDir(), "aliases.db"), false, false)
	primary, err := app.Context.GetBean[*slog.Logger](t.Context())
	if err != nil {
		t.Fatal(err)
	}
	alias, err := app.Context.GetBean[*slog.Logger](t.Context(), "defaultLogger")
	if err != nil || primary != alias {
		t.Fatal("primary/alias", err)
	}
	audit, err := app.Context.GetBean[*slog.Logger](t.Context(), "auditLogger")
	if err != nil || audit == primary {
		t.Fatal("logger isolation", err)
	}
	first, err := app.Context.GetBean[*service.Service](t.Context(), "orders")
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.Context.GetBean[*service.Service](t.Context(), "orderService")
	if err != nil || first != second {
		t.Fatal("service aliases", err)
	}
	if err = app.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if response := call(app.Router(), "GET", "/api/orders", ""); response.Code != 503 {
		t.Fatal(response.Code)
	}
}

func TestStartupRollbackClosesOpenedDatabase(t *testing.T) {
	var opened *infrastructure.Database
	injected := errors.New("controller failure")
	capture := func(_ context.Context, value any) (any, error) {
		opened = value.(*infrastructure.Database)
		return value, nil
	}
	refuse := func(_ context.Context, value any) (any, error) {
		return value, injected
	}
	_, err := application.Start(t.Context(), managed.WithProperties(map[string]string{configuration.DatabaseKey: filepath.Join(t.TempDir(), "rollback.db")}),
		managed.WithDecorator(bean.Decorator{
			Contract: gs.Contract("*github.com/longxi-go/ioc/examples/coffee/internal/infrastructure.Database"),
			Wrap:     capture,
		}),
		managed.WithDecorator(bean.Decorator{
			Contract: gs.Contract("*github.com/longxi-go/ioc/examples/coffee/internal/controller.API"),
			Wrap:     refuse,
		}))
	if !errors.Is(err, injected) || opened == nil || opened.SQL() == nil {
		t.Fatalf("rollback not exercised: %p %v", opened, err)
	}
	if err := opened.SQL().PingContext(t.Context()); err == nil {
		t.Fatal("startup rollback leaked database")
	}
}

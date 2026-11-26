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

package compiler

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedListenersUseExplicitDeclarationsAndPrivateMethods(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "explicit-listeners/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "listeners_test.go"), fixtureSource(t, "explicit-listeners/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated listeners: %v\n%s", err, out)
	}
}

func TestListenerDiagnosticsDoNotPublishPartialOutput(t *testing.T) {
	for _, tc := range []struct {
		name, fixture, diagnostic string
	}{
		{
			"non event",
			"listener-diagnostics/non-event.go.tmpl",
			"event.Event",
		},
		{
			"missing method",
			"listener-diagnostics/missing-method.go.tmpl",
			"OnEvent",
		},
		{
			"free function",
			"listener-diagnostics/free-function.go.tmpl",
			"method",
		},
		{
			"non struct",
			"listener-diagnostics/non-struct.go.tmpl",
			"struct",
		},
		{
			"orphan",
			"listener-diagnostics/orphan.go.tmpl",
			"owner",
		},
		{
			"wrong result",
			"listener-diagnostics/wrong-result.go.tmpl",
			"signature",
		},
		{
			"wrong context",
			"listener-diagnostics/wrong-context.go.tmpl",
			"signature",
		},
		{
			"variadic",
			"listener-diagnostics/variadic.go.tmpl",
			"signature",
		},
		{
			"request",
			"listener-diagnostics/request.go.tmpl",
			"singleton",
		},
		{
			"lazy all",
			"listener-diagnostics/lazy-all.go.tmpl",
			"eager",
		},
		{
			"legacy listener name",
			"listener-diagnostics/name-conflict.go.tmpl",
			"annotation",
		},
		{
			"method supplement",
			"listener-diagnostics/method-supplement.go.tmpl",
			"listen",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := fixture(t, fixtureSource(t, tc.fixture))
			if _, err := Run(context.Background(), Options{
				Dir: dir,
			}); err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
				t.Fatalf("want %q, got %v", tc.diagnostic, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "zz_ioc.gen.go")); !os.IsNotExist(err) {
				t.Fatal("invalid listener analysis published partial adapter", err)
			}
		})
	}
}

func TestGeneratedPublisherInjectionAndMultipleFactoryListenerOwners(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "publisher-owners/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "events_test.go"), fixtureSource(t, "publisher-owners/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("publisher/factory listener integration: %v\n%s", err, out)
	}
}

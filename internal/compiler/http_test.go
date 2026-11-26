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
	"strings"
	"testing"
)

func TestHTTPRequiresBackend(t *testing.T) {
	dir := fixture(t, `package main
// +ioc:controller
type Controller struct{}
// +ioc:http:get=/healthz
func (*Controller) health() {}
func main() {}
`)
	_, err := Run(context.Background(), Options{
		Dir: dir,
	})
	if err == nil || !strings.Contains(err.Error(), "HTTP backend") {
		t.Fatalf("expected HTTP backend diagnostic, got %v", err)
	}
}

func TestControllerWithoutEndpointsNeedsNoBackend(t *testing.T) {
	dir := fixture(t, "package main\n// +ioc:controller\ntype C struct{}\nfunc main() {}\n")
	report, err := Run(context.Background(), Options{
		Dir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Definitions) != 1 || report.Definitions[0].Name != "C" {
		t.Fatalf("controller registration: %+v", report)
	}
}

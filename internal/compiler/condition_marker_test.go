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
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConditionalMarkerSchema(t *testing.T) {
	for _, marker := range []string{
		"+ioc:conditional:on-property:prefix=storage,name=mode,havingValue=memory",
		"+ioc:conditional:on-bean",
		"+ioc:conditional:on-bean:type=Repository",
		"+ioc:conditional:on-bean:name=repository",
		"+ioc:conditional:on-bean:type=Repository,name=repository",
		"+ioc:conditional:on-missing-bean:type=Repository,name=repository",
	} {
		t.Run(marker, func(t *testing.T) {
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "config.go", "package p\n// "+marker+"\nfunc Bean() {}", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := readMarkers(fs, file.Comments[0], declarationTarget); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInvalidConditionalMarkersFailBeforeOutput(t *testing.T) {
	for _, tc := range []struct {
		marker string
		want   string
	}{
		{
			"+ioc:conditional:on-bean:type=Typo",
			"resolve condition type",
		},
		{
			"+ioc:conditional:on-bean:type=42",
			"must be a Go type",
		},
		{
			"+ioc:conditional:on-bean:type=Bean",
			"must be a Go type",
		},
		{
			"+ioc:conditional:on-bean:type=\"\"",
			"nonblank",
		},
		{
			"+ioc:conditional:on-bean:name=\" \"",
			"nonblank",
		},
		{
			"+ioc:conditional:on-bean:typo=string",
			"unknown argument",
		},
		{
			"+ioc:missing=Typo",
			"unknown or misplaced marker",
		},
		{
			"+ioc:present=Typo",
			"unknown or misplaced marker",
		},
	} {
		t.Run(tc.marker, func(t *testing.T) {
			source := "package main\n// +ioc:bean\n// " + tc.marker + "\nfunc Bean() int { return 1 }\nfunc main() {}\n"
			dir := fixture(t, source)
			_, err := Run(context.Background(), Options{
				Dir: dir,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "main.go:") {
				t.Fatalf("want %q with source, got %v", tc.want, err)
			}
		})
	}
}

func TestGeneratedConditionalMarkers(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "conditional-markers/main.go.tmpl"))
	for _, domain := range []string{"order", "payment"} {
		write(t, filepath.Join(dir, domain, "contracts", "repository.go"), "package contracts\ntype Repository interface { Name() string }\n")
	}
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "runtime_test.go"), fixtureSource(t, "conditional-markers/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated condition behavior: %v\n%s", err, output)
	}
}

func TestUninstantiatedGenericTypeReferencesAreRejected(t *testing.T) {
	for _, name := range []string{"Generic", "SliceAlias", "NamedAlias", "FunctionAlias", "Phantom"} {
		for _, marker := range []string{"+ioc:conditional:on-bean:type=", "+ioc:expose="} {
			t.Run(marker+name, func(t *testing.T) {
				source := strings.Replace(fixtureSource(t, "type-references/generic_alias.go.tmpl"), "TYPE_MARKER", marker+name, 1)
				dir := fixture(t, source)
				_, err := Run(context.Background(), Options{
					Dir: dir,
				})
				if err == nil || !strings.Contains(err.Error(), "requires generic type arguments") {
					t.Fatalf("accepted uninstantiated generic %s or misleading diagnostic: %v", name, err)
				}
			})
		}
	}
}

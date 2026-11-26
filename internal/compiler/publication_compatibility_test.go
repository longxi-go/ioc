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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortableLibraryAcceptsEquivalentConsumerBuilds(t *testing.T) {
	cases := []struct {
		name, variable, value string
		tags                  []string
	}{
		{
			name:     "cgo mode",
			variable: "CGO_ENABLED",
			value:    "0",
		},
		{
			name:     "unrelated build flags",
			variable: "GOFLAGS",
			value:    "-trimpath",
		},
		{
			name: "unused tag",
			tags: []string{"unused"},
		},
		{
			name:     "portable cross target",
			variable: "GOOS",
			value:    "windows",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CGO_ENABLED", "1")
			t.Setenv("GOFLAGS", "")
			library, path := libraryFixture(t)
			write(t, filepath.Join(library, "main.go"), fixtureSource(t, "portable-library/library.go.tmpl"))
			// Keep this fixture to one genuinely portable package.
			for _, dir := range []string{"internal", "private"} {
				if err := os.RemoveAll(filepath.Join(library, dir)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Run(context.Background(), Options{
				Dir:     library,
				Library: true,
			}); err != nil {
				t.Fatal(err)
			}
			before := mustRead(t, filepath.Join(library, publicationPath))
			adapter := mustRead(t, filepath.Join(library, generatedFileName))
			app := fixture(t, "package main\nfunc main() {}\n")
			write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
			if tc.variable != "" {
				t.Setenv(tc.variable, tc.value)
			}
			report, err := Run(context.Background(), Options{
				Dir:  app,
				Tags: tc.tags,
			})
			if err != nil || len(report.Definitions) != 1 {
				t.Fatalf("portable consumer rejected: %v %v", report, err)
			}
			if !bytes.Equal(before, mustRead(t, filepath.Join(library, publicationPath))) || !bytes.Equal(adapter, mustRead(t, filepath.Join(library, generatedFileName))) {
				t.Fatal("consumer modified library publication")
			}
		})
	}
}

func TestLibraryRejectsConsumerTypeShapeChanges(t *testing.T) {
	t.Setenv("GOARCH", "amd64")
	t.Setenv("CGO_ENABLED", "0")
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "portable-library/sized-library.go.tmpl"))
	for _, dir := range []string{"internal", "private"} {
		if err := os.RemoveAll(filepath.Join(library, dir)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main() {}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	t.Setenv("GOARCH", "386")
	_, err := Run(context.Background(), Options{
		Dir: app,
	})
	if err == nil || !strings.Contains(err.Error(), "incompatible build variant") {
		t.Fatalf("changed static type metadata accepted or unclear diagnostic: %v", err)
	}
	if _, err := os.Stat(filepath.Join(app, generatedFileName)); !os.IsNotExist(err) {
		t.Fatalf("incompatible library wrote consumer output: %v", err)
	}
}

func TestLibraryRejectsChangedDependencyTypesAtSameTarget(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "portable-library/dependency-sized-library.go.tmpl"))
	for _, dir := range []string{"internal", "private"} {
		if err := os.RemoveAll(filepath.Join(library, dir)); err != nil {
			t.Fatal(err)
		}
	}
	first, second := t.TempDir(), t.TempDir()
	for _, dir := range []string{first, second} {
		write(t, filepath.Join(dir, "go.mod"), "module example.org/shape\n\ngo 1.27.0\n")
	}
	write(t, filepath.Join(first, "shape.go"), "package shape\nconst Width = 8\n")
	write(t, filepath.Join(second, "shape.go"), "package shape\nconst Width = 4\n")
	appendDependency := func(dir, replacement string) {
		t.Helper()
		file := filepath.Join(dir, "go.mod")
		write(t, file, string(mustRead(t, file))+"require example.org/shape v1.0.0\nreplace example.org/shape => "+replacement+"\n")
	}
	appendDependency(library, first)
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main() {}\n")
	appendDependency(app, second)
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	_, err := Run(context.Background(), Options{
		Dir: app,
	})
	if err == nil || !strings.Contains(err.Error(), "incompatible build variant") {
		t.Fatalf("changed dependency result shape accepted: %v", err)
	}
}

func TestLibraryCompatibilityDoesNotLoadUnpublishedCommands(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "portable-library/library.go.tmpl"))
	for _, dir := range []string{"internal", "private"} {
		if err := os.RemoveAll(filepath.Join(library, dir)); err != nil {
			t.Fatal(err)
		}
	}
	first, second := t.TempDir(), t.TempDir()
	for _, dir := range []string{first, second} {
		write(t, filepath.Join(dir, "go.mod"), "module example.org/optional\n\ngo 1.27.0\n")
	}
	write(t, filepath.Join(first, "optional.go"), "package optional\nfunc OldAPI() {}\n")
	write(t, filepath.Join(second, "optional.go"), "package optional\n")
	write(t, filepath.Join(library, "cmd/unused/main.go"), fixtureSource(t, "portable-library/unpublished-command.go.tmpl"))
	file := filepath.Join(library, "go.mod")
	write(t, file, string(mustRead(t, file))+"require example.org/optional v1.0.0\nreplace example.org/optional => "+first+"\n")
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
		Roots:   []string{"."},
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, fixtureSource(t, "portable-library/consumer.go.tmpl"))
	file = filepath.Join(app, "go.mod")
	write(t, file, string(mustRead(t, file))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\nrequire example.org/optional v1.1.0\nreplace example.org/optional => "+second+"\n")
	if output, err := goOutput(context.Background(), app, "test", "."); err != nil {
		t.Fatalf("ordinary consumer fails: %v\n%s", err, output)
	}
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatalf("unpublished command affected a valid consumer: %v", err)
	}
}

func TestLibraryPublicationRespectsRaceBuildSelection(t *testing.T) {
	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOFLAGS", "-race")
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), "package library\n")
	for _, dir := range []string{"internal", "private"} {
		if err := os.RemoveAll(filepath.Join(library, dir)); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(library, "race.go"), fixtureSource(t, "portable-library/race.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main() {}\n")
	file := filepath.Join(app, "go.mod")
	write(t, file, string(mustRead(t, file))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	report, err := Run(context.Background(), Options{
		Dir: app,
	})
	if err != nil || len(report.Definitions) != 1 {
		t.Fatalf("matching race library rejected: %v %v", report, err)
	}
	t.Setenv("GOFLAGS", "")
	_, err = Run(context.Background(), Options{
		Dir: app,
	})
	if err == nil {
		t.Fatal("race-specific publication accepted without race build")
	}
}

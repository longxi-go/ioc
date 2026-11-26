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
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnnotationGeneratedEnvironmentAndConditions(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "annotation-generation/main.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "memory true" {
		t.Fatalf("generated annotation runtime: %v\n%s", err, out)
	}
	// Removing one name must make the multi-name condition fail even though the other matches.
	source := fixtureSource(t, "annotation-generation/main.go.tmpl")
	lines := strings.Split(source, "\n")
	removed := 0
	kept := lines[:0]
	for _, line := range lines {
		if strings.Contains(line, `"company.order.storage.second":`) {
			removed++
			continue
		}
		kept = append(kept, line)
	}
	if removed != 1 {
		t.Fatalf("expected exactly one property line, removed %d", removed)
	}
	source = strings.Join(kept, "\n")
	write(t, filepath.Join(dir, "main.go"), source)
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("go", "run", ".")
	cmd.Dir = dir
	if out, err = cmd.CombinedOutput(); err == nil {
		t.Fatalf("multi-name condition accepted missing property: %s", out)
	}
}

func TestAnnotationPublicationPreservesConditions(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "annotation-generation/library.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	data := mustRead(t, filepath.Join(library, "ioc/module.json"))
	var manifest publication
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, recipe := range manifest.Recipes {
		if recipe.Definition.Name != "published" {
			continue
		}
		found = true
		var raw map[string]json.RawMessage
		encoded, _ := json.Marshal(recipe)
		if err := json.Unmarshal(encoded, &raw); err != nil {
			t.Fatal(err)
		}
		var conditions []struct {
			Profiles []string
			Property *struct {
				Names       []string
				HavingValue *string
			}
		}
		if err := json.Unmarshal(raw["Conditions"], &conditions); err != nil {
			t.Fatalf("missing typed conditions: %v", err)
		}
		if len(conditions) != 3 ||
			strings.Join(conditions[2].Profiles, ",") != "dev,test" ||
			conditions[0].Property == nil ||
			strings.Join(conditions[0].Property.Names, ",") != "first,second" ||
			conditions[0].Property.HavingValue != nil ||
			conditions[1].Property == nil ||
			conditions[1].Property.HavingValue == nil ||
			*conditions[1].Property.HavingValue != "" {
			t.Fatalf("conditions lost in publication: %s", encoded)
		}

	}
	if !found {
		t.Fatal("published bean missing")
	}
	app := fixture(t, fixtureSource(t, "annotation-generation/consumer.go.tmpl"))
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = app
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("published adapters: %v\n%s", err, out)
	}
	for _, scenario := range []string{"profile", "property"} {
		cmd = exec.Command("go", "run", ".")
		cmd.Dir = app
		cmd.Env = append(os.Environ(), "SCENARIO="+scenario)
		if out, err := cmd.CombinedOutput(); err == nil {
			t.Fatalf("missing %s accepted: %s", scenario, out)
		}
	}
}

func TestAnnotationFactoryInterfaceListener(t *testing.T) {
	dir := fixture(t, fixtureSource(t, "annotation-generation/factory.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir: dir,
	}); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "1" {
		t.Fatalf("factory listener callback: %v\n%s", err, out)
	}
}

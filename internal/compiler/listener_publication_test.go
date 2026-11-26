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

	"github.com/longxi-go/ioc/pkg/bean"
)

func TestDependencyListenersPublishWithoutConsumerRegeneration(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "published-listeners/library.go.tmpl"))
	if _, err := Run(context.Background(), Options{
		Dir:     library,
		Library: true,
	}); err != nil {
		t.Fatal(err)
	}
	before := map[string]string{}
	if err := filepath.WalkDir(library, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			before[name] = string(mustRead(t, name))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	app := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(app, "events_test.go"), fixtureSource(t, "published-listeners/runtime_test.go.tmpl"))
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = app
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dependency event path: %v\n%s", err, out)
	}
	for name, content := range before {
		if string(mustRead(t, name)) != content {
			t.Fatalf("consumer changed dependency %s", name)
		}
	}
	manifestData := mustRead(t, filepath.Join(library, "ioc/module.json"))
	var manifest publication
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != schema ||
		manifest.RecipeABI != bean.SchemaVersion ||
		len(manifest.Recipes[0].Listeners) != 1 {
		t.Fatalf("listener publication metadata missing: %#v", manifest)
	}
	manifest.RecipeABI = 1
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(library, "ioc/module.json"), string(data))
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err == nil || !strings.Contains(err.Error(), "ABI") {
		t.Fatal("stale listener ABI accepted", err)
	}
}

func TestListenerOnlyDependencyRequiresPublishedAdapters(t *testing.T) {
	library, path := libraryFixture(t)
	write(t, filepath.Join(library, "main.go"), fixtureSource(t, "unpublished-listeners/library.go.tmpl"))
	app := fixture(t, "package main\nfunc main(){}\n")
	write(t, filepath.Join(app, "go.mod"), string(mustRead(t, filepath.Join(app, "go.mod")))+"require "+path+" v1.2.3\nreplace "+path+" => "+library+"\n")
	if _, err := Run(context.Background(), Options{
		Dir: app,
	}); err == nil || !strings.Contains(err.Error(), "generate the library") {
		t.Fatal("unpublished listener dependency ignored", err)
	}
}

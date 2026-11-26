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

package testmodule_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/longxi-go/ioc/internal/testmodule"
	"golang.org/x/mod/modfile"
)

func ExampleWrite() {
	dir, err := os.MkdirTemp("", "ioc-consumer-*")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			panic(err)
		}
	}()
	if err := testmodule.Write(dir, "example.org/application", "../.."); err != nil {
		panic(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		panic(err)
	}
	module, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(module.Module.Mod.Path)
	fmt.Println(module.Replace[0].Old.Path)
	// Output:
	// example.org/application
	// github.com/longxi-go/ioc
}

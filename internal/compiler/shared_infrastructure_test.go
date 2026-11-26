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
	"errors"
	"go/token"
	"testing"

	"github.com/longxi-go/gosr/pkg/codegen/diagnostic"
	"golang.org/x/tools/go/packages"
)

func TestEmitterRejectsConflictingAliases(t *testing.T) {
	e := newEmitter("example/app")
	e.use("example/a", "same")
	e.use("example/b", "same")
	if _, err := e.finish("app", ""); err == nil {
		t.Fatal("conflicting import aliases accepted")
	}
}

func TestAtPreservesStructuredPositionAndCause(t *testing.T) {
	fs := token.NewFileSet()
	file := fs.AddFile("app.go", -1, 100)
	cause := errors.New("invalid bean")
	err := at(&packages.Package{
		Fset: fs,
	}, file.Pos(5), cause)
	var d *diagnostic.Error
	if !errors.As(err, &d) || !errors.Is(err, cause) || d.Position.Filename != "app.go" {
		t.Fatal(err)
	}
}

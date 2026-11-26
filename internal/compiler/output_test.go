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
	"path/filepath"
	"testing"
)

func TestOutputPreflightPreservesEarlierFilesOnPathConflict(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.go")
	last := filepath.Join(dir, "z.go")
	before := generatedHeader + "package fixture\n"
	write(t, first, before)
	if err := os.Mkdir(last, 0o755); err != nil {
		t.Fatal(err)
	}
	err := commitOutputs(context.Background(), map[string][]byte{first: []byte(generatedHeader + "package changed\n"), last: []byte(generatedHeader + "package fixture\n")}, nil)
	if err == nil {
		t.Fatal("accepted directory destination")
	}
	if string(mustRead(t, first)) != before {
		t.Fatal("partial publication after predictable path conflict")
	}
}

func TestOutputRefusesUnownedSourceAndCleansOnlyOwnedSelection(t *testing.T) {
	dir := t.TempDir()
	foreign := filepath.Join(dir, "zz_ioc.gen.go")
	write(t, foreign, "package user\n")
	if err := commitOutputs(context.Background(), map[string][]byte{foreign: []byte(generatedHeader + "package generated\n")}, nil); err == nil {
		t.Fatal("overwrote user file")
	}
	stale := filepath.Join(dir, "stale.go")
	write(t, stale, generatedHeader+"package user\n")
	if err := commitOutputs(context.Background(), nil, map[string][]byte{stale: nil}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale file retained")
	}
	if string(mustRead(t, foreign)) != "package user\n" {
		t.Fatal("unselected source changed")
	}
}

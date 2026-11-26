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

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/longxi-go/ioc/internal/testmodule"
)

func TestUsageAndFlags(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{
			"wat",
		},
		{
			"generate",
			"--wat",
		},
		{
			"generate",
			"extra",
		},
	} {
		var out, errout bytes.Buffer
		if code := runCLI(context.Background(), args, &out, &errout); code == 0 {
			t.Fatalf("accepted %v", args)
		}
	}
	var out, errout bytes.Buffer
	if code := runCLI(context.Background(), []string{"help"}, &out, &errout); code != 0 || !strings.Contains(out.String(), "generate") {
		t.Fatalf("help: %d %s", code, &out)
	}
}

func TestGenerateCheckExplainAndGraph(t *testing.T) {
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := testmodule.Write(dir, "example.org/commands", root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/commands/main.go.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, verb := range []string{"explain", "graph", "generate", "check", "explain", "graph"} {
		var out, errout bytes.Buffer
		if code := runCLI(context.Background(), []string{verb, "--dir", dir}, &out, &errout); code != 0 {
			t.Fatalf("%s: %d %s", verb, code, &errout)
		}
		if out.Len() == 0 {
			t.Fatalf("%s produced no output", verb)
		}
		if verb == "explain" && !strings.Contains(out.String(), "Service") {
			t.Fatal("missing candidate metadata")
		}
		if verb == "graph" &&
			(!strings.Contains(out.String(), "direct") || !strings.Contains(out.String(), "dependency")) {
			t.Fatal("missing dependency edges")
		}
	}
	if code := runCLI(context.Background(), []string{"help"}, failingWriter{}, io.Discard); code != 1 {
		t.Fatal("ignored output failure", code)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := runCLI(ctx, []string{"explain", "--dir", dir}, io.Discard, io.Discard); code != 1 {
		t.Fatal("ignored cancellation", code)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("writer failed")
}

func runCLI(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return Run(ctx, args, stdout, stderr, Options{})
}

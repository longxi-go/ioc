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
	"slices"
	"testing"
)

func TestFlagWordsPreserveGoQuoting(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{
			`'-tags=one two' -race`,
			[]string{"-tags=one two", "-race"},
		},
		{
			`"-modfile=/path with space/go.mod"`,
			[]string{"-modfile=/path with space/go.mod"},
		},
		{
			`-ldflags='literal'`,
			[]string{"-ldflags='literal'"},
		},
		{
			`'' -race`,
			[]string{"", "-race"},
		},
		{
			`-ldflags=a'b"c`,
			[]string{`-ldflags=a'b"c`},
		},
	}
	for _, tc := range cases {
		got, err := splitFlagWords(tc.input)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Fatalf("words=%q want=%q err=%v", got, tc.want, err)
		}
		joined, err := joinFlagWords(got)
		if err != nil {
			t.Fatal(err)
		}
		again, err := splitFlagWords(joined)
		if err != nil || !slices.Equal(again, tc.want) {
			t.Fatalf("roundtrip=%q want=%q err=%v", again, tc.want, err)
		}
	}
}

func TestFlagWordsRejectUnterminatedQuote(t *testing.T) {
	if _, err := splitFlagWords(`'-tags=broken`); err == nil {
		t.Fatal("unterminated quote accepted")
	}
	if _, err := buildFlagsOnly(`'-tags=broken`); err == nil {
		t.Fatal("invalid build flags accepted")
	}
	if _, err := effectiveTags(nil, `'-tags=broken`); err == nil {
		t.Fatal("invalid effective tags accepted")
	}
}

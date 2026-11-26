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

func TestEffectiveTagsIncludeEnabledInstrumentation(t *testing.T) {
	cases := []struct {
		name, flags    string
		explicit, want []string
	}{
		{
			name:  "race",
			flags: "-race",
			want:  []string{"race"},
		},
		{
			name:  "disabled race",
			flags: "-race=false",
		},
		{
			name:  "memory sanitizer",
			flags: "-msan=true",
			want:  []string{"msan"},
		},
		{
			name:  "address sanitizer",
			flags: "-asan",
			want:  []string{"asan"},
		},
		{
			name:     "explicit tags override flag tags",
			flags:    "-tags=old -race=true",
			explicit: []string{"special"},
			want:     []string{"special", "race"},
		},
		{
			name:  "flag tags and instrumentation",
			flags: "-tags=first,second -race",
			want:  []string{"first", "second", "race"},
		},
		{
			name:  "quoted flag with space separated tags",
			flags: "'-tags=first second' -race",
			want:  []string{"first", "second", "race"},
		},
		{
			name:  "legacy quoted tag value",
			flags: "-tags='special'",
			want:  []string{"special"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := effectiveTags(tc.explicit, tc.flags); err != nil || !slices.Equal(got, tc.want) {
				t.Fatalf("tags=%v want=%v err=%v", got, tc.want, err)
			}
		})
	}
}

func TestBuildFlagsPreserveQuotedTagWords(t *testing.T) {
	want := "'-tags=first second' -race"
	if got, err := buildFlagsOnly("-mod=readonly " + want); err != nil || got != want {
		t.Fatalf("build flags=%q want=%q err=%v", got, want, err)
	}
}

func TestBuildFlagsRequireFlagPrefix(t *testing.T) {
	if got, err := buildFlagsOnly("tags race msan asan -race"); err != nil || got != "-race" {
		t.Fatalf("non-flag words retained: %q, %v", got, err)
	}
}

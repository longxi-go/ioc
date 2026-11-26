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

import "testing"

func TestFilenameConstraintFollowsGoSuffixGrammar(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{
			"repository_linux.go",
			"linux",
		},
		{
			"repository_linux_amd64.go",
			"linux && amd64",
		},
		{
			"repository_amd64_linux.go",
			"linux",
		},
		{
			"repository_386_amd64.go",
			"amd64",
		},
		{
			"repository.extra_linux.go",
			"",
		},
		{
			"_ignored_linux.go",
			"",
		},
		{
			".ignored_linux.go",
			"",
		},
		{
			"repository_unknown.go",
			"",
		},
		{
			"linux.go",
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filenameConstraint(tc.name); got != tc.want {
				t.Fatalf("constraint=%q want=%q", got, tc.want)
			}
		})
	}
}

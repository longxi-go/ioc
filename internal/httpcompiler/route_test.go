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

package httpcompiler

import "testing"

func TestPath(t *testing.T) {
	for _, tt := range []struct {
		base, endpoint, want string
	}{
		{
			"/api/",
			"/ready/",
			"/api/ready/",
		},
		{
			"/api",
			"/",
			"/api/",
		},
		{
			"",
			"/ready",
			"/ready",
		},
		{
			"/",
			"/ready",
			"/ready",
		},
		{
			"/api/",
			"",
			"/api/",
		},
	} {
		got, err := Path(tt.base, tt.endpoint)
		if err != nil || got != tt.want {
			t.Errorf("%q + %q = %q, %v", tt.base, tt.endpoint, got, err)
		}
	}
	for _, path := range []string{"relative", "/a//b", "/a/../b", "/./b", "/a?x=y", "/a#b", "/a%2fb", "/a\\b", "/${path}", "/a b", "/a\x00"} {
		if _, err := Path("", path); err == nil {
			t.Errorf("accepted ambiguous path %q", path)
		}
	}
}

func TestMethod(t *testing.T) {
	for _, method := range []string{"GET", "M-SEARCH", "CUSTOM", "custom"} {
		if err := Method(method); err != nil {
			t.Errorf("valid HTTP token %q: %v", method, err)
		}
	}
	for _, method := range []string{"", "GET POST", "GET\n", "${method}", "GET/POST", "méthod"} {
		if err := Method(method); err == nil {
			t.Errorf("accepted invalid method %q", method)
		}
	}
}

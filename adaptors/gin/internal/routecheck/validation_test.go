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

package routecheck

import "testing"

func TestNativeDeclarations(t *testing.T) {
	for _, item := range []struct {
		method, path string
		valid        bool
	}{
		{
			"GET",
			"/",
			true,
		},
		{
			"PURGE",
			"/cache/:key",
			true,
		},
		{
			"TRACE",
			"/x/*tail",
			true,
		},
		{
			"CONNECT",
			"/prefix:name",
			true,
		},
		{
			"GET",
			"/literal\\:colon",
			true,
		},
		{
			"",
			"/",
			false,
		},
		{
			"get",
			"/",
			false,
		},
		{
			"M-SEARCH",
			"/",
			false,
		},
		{
			"GET",
			"relative",
			false,
		},
		{
			"GET",
			"/:a:b",
			false,
		},
		{
			"GET",
			"/*tail/more",
			false,
		},
		{
			"GET",
			"/prefix*tail",
			false,
		},
		{
			"GET",
			"/:",
			false,
		},
		{
			"GET",
			"/bad\\escape",
			false,
		},
	} {
		err := Validate(item.method, item.path)
		if (err == nil) != item.valid {
			t.Fatalf("%s %s: %v", item.method, item.path, err)
		}
	}
}

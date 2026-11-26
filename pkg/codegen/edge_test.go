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

package codegen

import (
	"encoding/json"
	"testing"
)

func TestEdgeKindsPreserveGraphJSON(t *testing.T) {
	tests := []struct {
		kind EdgeKind
		wire string
	}{
		{
			DependencyEdge,
			`"dependency"`,
		},
		{
			PrerequisiteEdge,
			`"depends-on"`,
		},
	}
	for _, test := range tests {
		encoded, err := json.Marshal(Edge{
			Kind: test.kind,
		})
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if string(fields["Kind"]) != test.wire {
			t.Fatalf("graph kind changed: got %s, want %s", fields["Kind"], test.wire)
		}
		var decoded Edge
		if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.Kind != test.kind {
			t.Fatalf("edge round trip: %#v %v", decoded, err)
		}
	}
}

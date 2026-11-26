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
	"cmp"
)

func edgeLess(a, b Edge) bool {
	return cmp.Or(
		cmp.Compare(a.From, b.From),
		cmp.Compare(a.To, b.To),
		cmp.Compare(a.Kind, b.Kind),
		cmp.Compare(a.Access, b.Access),
		cmp.Compare(a.Source.Package, b.Source.Package),
		cmp.Compare(a.Source.File, b.Source.File),
		cmp.Compare(a.Source.Line, b.Source.Line),
	) < 0
}

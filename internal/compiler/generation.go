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

// generation is owned by one invocation. Typed state keeps phase dependencies
// explicit without a string-keyed context bag or globally registered plugins.
type generation struct {
	options                 Options
	dir                     string
	module                  module
	target                  target
	patterns                []string
	overlay, outputs, owned map[string][]byte
	models, local           []*packageModel
	recipes                 []publishedRecipe
	indexes                 []string
	entry                   *packageModel
	report                  *Report
	validated               bool
}

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

import "net/http"

// Shortcuts is the closed supported shorthand vocabulary. Each call owns its map.
func Shortcuts() map[string]string {
	return map[string]string{
		"http:get":     http.MethodGet,
		"http:post":    http.MethodPost,
		"http:put":     http.MethodPut,
		"http:patch":   http.MethodPatch,
		"http:delete":  http.MethodDelete,
		"http:head":    http.MethodHead,
		"http:options": http.MethodOptions,
	}
}

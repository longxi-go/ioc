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

import "errors"

const (
	invalidMethodMessage = "gin requires an uppercase alphabetic method"
)

// Validate applies declaration-local Gin constraints without changing its global
// mode or log output. Cross-route tree conflicts require native assembly.
func Validate(method, path string) error {
	if method == "" {
		return errors.New(invalidMethodMessage)
	}
	for _, c := range method {
		if c < 'A' || c > 'Z' {
			return errors.New(invalidMethodMessage)
		}
	}
	return validatePath(path)
}

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

import (
	"fmt"
	"strings"
)

const (
	invalidMethodFormat    = "invalid HTTP method %q"
	methodTokenPunctuation = "!#$%&'*+-.^_`|~"
)

// Method requires an explicit RFC HTTP token. Method names are case-sensitive;
// custom methods remain available for the selected backend to validate.
func Method(method string) error {
	if method == "" {
		return fmt.Errorf(invalidMethodFormat, method)
	}
	for _, r := range method {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune(methodTokenPunctuation, r) {
			continue
		}
		return fmt.Errorf(invalidMethodFormat, method)
	}
	return nil
}

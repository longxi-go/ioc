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

import (
	"errors"
	"strings"
)

const (
	absolutePathMessage    = "path must begin with /"
	invalidEscapeMessage   = "invalid wildcard escape"
	invalidWildcardMessage = "wildcard requires one nonempty name per segment"
	invalidCatchAllMessage = "catch-all must follow / and terminate the path"
)

// validatePath checks declaration-local wildcard constraints without touching
// Gin's global logging or mode. Native tree conflicts remain an assembly concern.
func validatePath(path string) error {
	if !strings.HasPrefix(path, "/") {
		return errors.New(absolutePathMessage)
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		for j := 0; j < len(segment); j++ {
			c := segment[j]
			if c == '\\' {
				if j+1 == len(segment) || segment[j+1] != ':' {
					return errors.New(invalidEscapeMessage)
				}
				j++
				continue
			}
			if c != ':' && c != '*' {
				continue
			}
			name := segment[j+1:]
			if name == "" || strings.ContainsAny(name, ":*") {
				return errors.New(invalidWildcardMessage)
			}
			if c == '*' && (j != 0 || i != len(segments)-1) {
				return errors.New(invalidCatchAllMessage)
			}
			break
		}
	}
	return nil
}

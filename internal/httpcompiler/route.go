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
	"unicode"

	env "github.com/longxi-go/gosr/pkg/environment"
)

// Marker and Controller identify the only HTTP declaration protocol understood by
// core; concrete transport rules such as wildcard matching belong to adapters.
const (
	Controller              = "controller"
	Marker                  = "http:route"
	Prefix                  = "http:"
	invalidPathFormat       = "invalid HTTP path %q"
	pathSeparator           = "/"
	repeatedPathSeparator   = pathSeparator + pathSeparator
	forbiddenPathCharacters = "?#%\\"
	currentPathSegment      = "."
	parentPathSegment       = ".."
)

// MarkerValue lets controller-tools retain strict property and duplicate checks.
type MarkerValue struct {
	Method string `marker:"method,optional"`
	Path   string `marker:"path"`
}

// Path joins boundary slashes without cleaning meaningful endpoint trailing slashes
// or silently accepting ambiguous paths. Empty base denotes a standalone handler.
func Path(base, endpoint string) (string, error) {
	for _, p := range []string{base, endpoint} {
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, pathSeparator) || strings.Contains(p, repeatedPathSeparator) || strings.ContainsAny(p, forbiddenPathCharacters) || strings.Contains(p, env.PlaceholderPrefix) || strings.IndexFunc(p, unicode.IsSpace) >= 0 || strings.IndexFunc(p, unicode.IsControl) >= 0 {
			return "", fmt.Errorf(invalidPathFormat, p)
		}
		for _, segment := range strings.Split(p, pathSeparator) {
			if segment == currentPathSegment || segment == parentPathSegment {
				return "", fmt.Errorf(invalidPathFormat, p)
			}
		}
	}
	if endpoint == "" {
		return base, nil
	}
	if base == "" || base == pathSeparator {
		return endpoint, nil
	}
	return strings.TrimRight(base, pathSeparator) + pathSeparator + strings.TrimLeft(endpoint, pathSeparator), nil
}

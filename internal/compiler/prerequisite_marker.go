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
	"fmt"
	"strings"
)

const (
	dependsOnRequiresDistinctNonblankBeanNamesFormat = "depends-on requires distinct nonblank Bean names: %q"
)

func compilePrerequisiteNames(a annotation) ([]string, error) {
	names := a.strings(markerDependsOn)
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" || seen[name] {
			return nil, fmt.Errorf(dependsOnRequiresDistinctNonblankBeanNamesFormat, name)
		}
		seen[name] = true
	}
	return names, nil
}

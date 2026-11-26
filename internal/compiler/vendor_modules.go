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
	"path/filepath"
	"strings"
)

// vendorModules includes a header only after a package establishes source ownership.
// Explicit requirements and replacement summaries alone need no source directory.
func vendorModules(data []byte, vendorRoot string) []module {
	var result []module
	var candidate module
	for line := range strings.SplitSeq(string(data), sourceLineSeparator) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == vendorModulePrefix {
			candidate = module{}
			if len(fields) >= vendorVersionHeaderFields && strings.HasPrefix(fields[2], moduleVersionPrefix) {
				candidate = module{
					Path:    fields[1],
					Version: fields[2],
					Dir:     filepath.Join(vendorRoot, vendorDirectory, filepath.FromSlash(fields[1])),
				}
			}
			continue
		}
		// Explicit requirements and replacement summaries may have no source.
		// Only package lines establish that a module participates in vendor.
		if len(fields) == 1 && !strings.HasPrefix(fields[0], vendorModulePrefix) && candidate.Path != "" {
			result = append(result, candidate)
			candidate = module{}
		}
	}
	return result
}

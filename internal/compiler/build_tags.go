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
	"slices"
	"strconv"
	"strings"
)

const (
	moduleFlagLeadingQuote = " '"
)

func buildFlagsOnly(flags string) (string, error) {
	words, err := splitFlagWords(flags)
	if err != nil {
		return "", err
	}
	var result []string
	for _, flag := range words {
		name, _, _ := strings.Cut(flag, goFlagAssignment)
		switch name {
		case goFlagPrefix + buildTagsName, goFlagPrefix + raceBuildTag, goFlagPrefix + msanBuildTag, goFlagPrefix + asanBuildTag:
			result = append(result, flag)
		}
	}
	return joinFlagWords(result)
}

// effectiveTags mirrors source-selection tags added by Go's instrumentation
// flags as well as explicit build tags. Explicit -tags replaces GOFLAGS -tags,
// but must not erase implicit race, msan or asan selection.
func effectiveTags(explicit []string, flags string) ([]string, error) {
	words, err := splitFlagWords(flags)
	if err != nil {
		return nil, err
	}
	tags := slices.Clone(explicit)
	instrumentation := map[string]bool{}
	for _, flag := range words {
		name, value, assigned := strings.Cut(strings.TrimPrefix(flag, goFlagPrefix), goFlagAssignment)
		switch name {
		case buildTagsName:
			if len(explicit) == 0 {
				// Go retains legacy quoted/space-separated tag values while
				// comma-separated values use the current flag spelling.
				if strings.ContainsAny(value, moduleFlagLeadingQuote) {
					tags, err = splitFlagWords(value)
					if err != nil {
						return nil, err
					}
				} else {
					tags = strings.FieldsFunc(value, splitBuildTag)
				}
			}
		case raceBuildTag, msanBuildTag, asanBuildTag:
			enabled := !assigned
			if assigned {
				parsed, err := strconv.ParseBool(value)
				enabled = err == nil && parsed
			}
			instrumentation[name] = enabled
		}
	}
	for _, name := range []string{raceBuildTag, msanBuildTag, asanBuildTag} {
		if instrumentation[name] && !slices.Contains(tags, name) {
			tags = append(tags, name)
		}
	}
	return tags, nil
}

func splitBuildTag(r rune) bool {
	return r == rune(buildTagSeparator[0])
}

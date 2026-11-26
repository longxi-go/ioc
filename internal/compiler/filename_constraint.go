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
	"go/build"
	"io"
	"strings"
)

const (
	probeGOOS           = "ioc_unknown_os"
	probeGOARCH         = "ioc_unknown_arch"
	platformProbeName   = "probe"
	platformProbePrefix = platformProbeName + goFilenameSeparator
	platformProbeSource = packageClausePrefix + platformProbeName + "\n"
)

type platformKind uint8

const (
	unknownPlatform platformKind = iota
	osPlatform
	archPlatform
)

// filenameConstraint mirrors suffix order but delegates recognized platform
// names to go/build. It does not maintain a second toolchain platform table.
func filenameConstraint(name string) string {
	if ignoredGoFilename(name) {
		return ""
	}
	stem, _, _ := strings.Cut(name, goFilenameDot)
	first := strings.Index(stem, goFilenameSeparator)
	if first < 0 {
		return ""
	}
	parts := strings.Split(stem[first:], goFilenameSeparator)
	n := len(parts)
	last := parts[n-1]
	if classifyPlatform(last) == archPlatform {
		if n > 1 && classifyPlatform(parts[n-2]) == osPlatform {
			return parts[n-2] + buildConstraintConjunction + last
		}
		return last
	}
	if classifyPlatform(last) == osPlatform {
		return last
	}
	return ""
}

func classifyPlatform(tag string) platformKind {
	if tag == "" {
		return unknownPlatform
	}
	probe := build.Context{
		GOOS:     probeGOOS,
		GOARCH:   probeGOARCH,
		OpenFile: openPlatformProbe,
	}
	name := platformProbePrefix + tag + goSourceExtension
	if matches, _ := probe.MatchFile(currentDirectory, name); matches {
		return unknownPlatform
	}
	// Only a recognized OS followed by an architecture creates a pair. With
	// an unknown GOOS, that pair is rejected; an arch_arch suffix is not a pair.
	probe.GOARCH = build.Default.GOARCH
	name = platformProbePrefix + tag + goFilenameSeparator + probe.GOARCH + goSourceExtension
	matches, _ := probe.MatchFile(currentDirectory, name)
	if !matches {
		return osPlatform
	}
	return archPlatform
}

func openPlatformProbe(string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(platformProbeSource)), nil
}

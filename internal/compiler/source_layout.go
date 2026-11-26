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

import "strings"

// Go filename grammar and source-tree names belong to discovery and isolation,
// rather than the IoC publication layout or the portable Gosr contracts.
const (
	internalDirectory     = "internal"
	relativePackagePrefix = "./"
	goFilenameDot         = "."
	goFilenameSeparator   = "_"
	goSourceExtension     = goFilenameDot + "go"
	goTestSuffix          = goFilenameSeparator + "test" + goSourceExtension
	packageClausePrefix   = "package "
	currentDirectory      = "."
	vendorDirectory       = "vendor"
	vendorModulesFile     = "modules.txt"
	testdataDirectory     = "testdata"
	testsDirectory        = "tests"
)

func ignoredGoFilename(name string) bool {
	return strings.HasPrefix(name, goFilenameDot) || strings.HasPrefix(name, goFilenameSeparator)
}

func nonTestGoSource(name string) bool {
	return !ignoredGoFilename(name) && strings.HasSuffix(name, goSourceExtension) &&
		!strings.HasSuffix(name, goTestSuffix)
}

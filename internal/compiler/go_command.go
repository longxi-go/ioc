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

// Command tokens and module environment keys must agree between discovery,
// source loading and read-only isolation. These are tooling policy, not Gosr API.
const (
	goExecutable         = "go"
	goListCommand        = "list"
	goEnvCommand         = "env"
	goModCommand         = "mod"
	goDownloadCommand    = "download"
	jsonFlag             = "-json"
	moduleFlag           = "-m"
	moduleModePrefix     = "-mod="
	vendorModuleMode     = "vendor"
	moduleFileFlagPrefix = "-modfile="
	goFlagPrefix         = "-"
	goFlagAssignment     = "="
	buildTagsName        = "tags"
	raceBuildTag         = "race"
	msanBuildTag         = "msan"
	asanBuildTag         = "asan"
	buildTagsFlagPrefix  = goFlagPrefix + buildTagsName + goFlagAssignment
	goOSKey              = "GOOS"
	goArchKey            = "GOARCH"
	cgoEnabledKey        = "CGO_ENABLED"
	goFlagsKey           = "GOFLAGS"
	goWorkKey            = "GOWORK"
	goWorkOff            = "off"
)

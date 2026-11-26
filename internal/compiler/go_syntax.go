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

// Tokens shared by emitted Go expressions and canonical contract identities.
// Domain identities and build constraints retain their separate grammars.
const (
	goLanguageVersionPrefix = "go"
	goSelector              = "."
	goPointer               = "*"
	goSlice                 = "[]"
	goTypeArgumentsOpen     = "["
	goTypeArgumentsClose    = "]"
	goMapOpen               = "map["
	goChannel               = "chan "
	goSendChannel           = "chan<- "
	goReceiveChannel        = "<-chan "
	goTypeSeparator         = ","
	goTypeSpace             = " "
	goMemberSeparator       = ";"
	goStructOpen            = "struct{"
	goInterfaceOpen         = "interface{"
	goBodyClose             = "}"
	goUnderlyingType        = "~"
	goUnionSeparator        = "|"
	goVariadic              = "..."
	goFunctionOpen          = "func("
	goResultsOpen           = ")("
	goCallClose             = ")"
	goArgumentSeparator     = ", "
)

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
	"github.com/longxi-go/gosr/pkg/codegen/render"
	protocol "github.com/longxi-go/ioc/pkg/codegen/http"
)

// Bootstrap supplies only typed declarations for the initial source overlay.
// User startup code may reference generated aggregators before the first run;
// root replaces these in-memory declarations with fully validated final output.
func Bootstrap(p protocol.Package) ([]byte, error) {
	set, err := render.ParseFS(templates, nil, "templates/bootstrap.go.tmpl")
	if err != nil {
		return nil, err
	}
	return set.Execute("bootstrap.go.tmpl", packageView{
		Name: p.Name,
	})
}

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
	"go/token"
	"path/filepath"

	"github.com/longxi-go/gosr/pkg/codegen/diagnostic"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"golang.org/x/tools/go/packages"
)

func source(pkg *packages.Package, pos token.Pos) gs.Source {
	p := pkg.Fset.Position(pos)
	return gs.Source{
		Package: pkg.PkgPath,
		File:    filepath.Base(p.Filename),
		Line:    p.Line,
	}
}

func at(pkg *packages.Package, pos token.Pos, err error) error {
	return diagnostic.Wrap(stageAnalyze, pkg.Fset.Position(pos), "", err)
}

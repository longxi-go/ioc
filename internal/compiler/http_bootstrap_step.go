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
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/longxi-go/gosr/tools/pkg/packageload"
	http "github.com/longxi-go/ioc/pkg/codegen/http"
	"golang.org/x/tools/go/packages"
)

const (
	httpBootstrapFormat        = "HTTP declarations %s: %w"
	httpBootstrapUnownedFormat = "HTTP bootstrap cannot overlay unowned source %s"
)

// bootstrapHTTP makes backend entry points available before typed analysis while
// preserving the same module, source selection and build environment.
func bootstrapHTTP(ctx context.Context, g *generation) error {
	backend, ok := g.options.HTTP.(http.Bootstrapper)
	if !ok {
		return nil
	}
	request := packageLoad{
		Dir:      g.dir,
		Tags:     g.options.Tags,
		Patterns: g.patterns,
		Overlay:  g.overlay,
	}
	cfg := packageConfig(ctx, request)
	cfg.Mode = packages.NeedName | packages.NeedFiles | packages.NeedModule
	pkgs, err := packageload.Load(cfg, request.Patterns...)
	if err != nil {
		return err
	}
	if err := packageload.Errors(phaseTypeChecking, pkgs); err != nil {
		return err
	}
	g.bootstrapOverlay = map[string][]byte{}
	for _, pkg := range pkgs {
		if pkg.Module == nil || pkg.Module.Path != g.module.Path || len(pkg.GoFiles) == 0 {
			continue
		}
		data, err := backend.Bootstrap(http.Package{
			Path: pkg.PkgPath,
			Name: pkg.Name,
		})
		if err != nil {
			return fmt.Errorf(httpBootstrapFormat, pkg.PkgPath, err)
		}
		if len(data) == 0 {
			continue
		}
		path := filepath.Join(filepath.Dir(pkg.GoFiles[0]), generatedFileName)
		if err := httpBootstrapDestination(path); err != nil {
			return err
		}
		g.bootstrapOverlay[path] = g.overlay[path]
		g.overlay[path] = append([]byte(generatedHeader), data...)
	}
	return nil
}

// restoreHTTPOverlay prevents temporary declarations from satisfying missing or
// ill-typed final output, including packages that emit no HTTP artifact.
func restoreHTTPOverlay(g *generation) {
	for path, original := range g.bootstrapOverlay {
		if original == nil {
			delete(g.overlay, path)
		} else {
			g.overlay[path] = original
		}
	}
	g.bootstrapOverlay = nil
}

// Bootstrap has the same ownership boundary as final output. Checking before
// overlay installation prevents read-only inspection from hiding real source.
func httpBootstrapDestination(path string) error {
	current, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !ownsOutput(path, current) {
		return fmt.Errorf(httpBootstrapUnownedFormat, path)
	}
	return nil
}

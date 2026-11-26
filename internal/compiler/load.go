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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/longxi-go/gosr/tools/pkg/packageload"
	"golang.org/x/tools/go/packages"
)

const (
	vendorVersionHeaderFields  = 3
	allModulesArgument         = "all"
	buildTagSeparator          = ","
	goArgumentDisplaySeparator = " "
	goCommandFailureFormat     = "go %s: %w: %s"
	moduleVersionPrefix        = "v"
	moduleVersionSeparator     = "@"
	vendorModulePrefix         = "#"

	noSelectedModuleOwnsFormat   = "no selected module owns %s"
	phaseTypeChecking            = "type checking"
	selectedModuleDownloadFormat = "selected module download: %w"
	vendorModuleListFormat       = "vendor module list: %w"
)

type module struct {
	Path      string
	Dir       string
	GoMod     string
	Version   string
	GoVersion string
	Main      bool
	Replace   *module
}

type target struct {
	GOOS        string
	GOARCH      string
	CGO_ENABLED string
	GOFLAGS     string
}

func goOutput(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, goExecutable, args...)
	cmd.Dir = dir
	cmd.Env = commandEnvironment(ctx)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf(goCommandFailureFormat, strings.Join(args, goArgumentDisplaySeparator), err, stderr.String())
	}
	return b, nil
}

func loadModule(ctx context.Context, dir string) (module, error) {
	b, err := goOutput(ctx, dir, goListCommand, moduleFlag, jsonFlag)
	if err != nil {
		return module{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	var selected module
	for {
		var m module
		if err := decoder.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return module{}, err
		}
		rel, e := filepath.Rel(m.Dir, dir)
		if e == nil &&
			safeRelative(rel) &&
			(selected.Dir == "" || len(m.Dir) > len(selected.Dir)) {
			selected = m
		}
	}
	if selected.Dir == "" {
		return module{}, fmt.Errorf(noSelectedModuleOwnsFormat, dir)
	}
	return selected, nil
}

func selectedModules(ctx context.Context, dir string) ([]module, error) {
	flags, err := goOutput(ctx, dir, goEnvCommand, goFlagsKey)
	if err != nil {
		return nil, err
	}
	mode := ""
	words, err := splitFlagWords(string(flags))
	if err != nil {
		return nil, err
	}
	for _, flag := range words {
		if value, found := strings.CutPrefix(flag, moduleModePrefix); found {
			mode = value
		}
	}
	vendorRoot := dir
	workBytes, err := goOutput(ctx, dir, goEnvCommand, goWorkKey)
	if err != nil {
		return nil, err
	}
	workPath := strings.TrimSpace(string(workBytes))
	if workPath != "" && workPath != goWorkOff {
		vendorRoot = filepath.Dir(workPath)
	}
	vendorFile := filepath.Join(vendorRoot, vendorDirectory, vendorModulesFile)
	_, vendorErr := os.Stat(vendorFile)
	vendor := mode == vendorModuleMode || (mode == "" && vendorErr == nil)
	if vendor {
		data, err := os.ReadFile(vendorFile)
		if err != nil {
			return nil, fmt.Errorf(vendorModuleListFormat, err)
		}
		result := vendorModules(data, vendorRoot)

		if workPath != "" && workPath != goWorkOff {
			data, err := goOutput(ctx, dir, goListCommand, moduleFlag, jsonFlag)
			if err != nil {
				return nil, err
			}
			decoder := json.NewDecoder(bytes.NewReader(data))
			for {
				var main module
				if err := decoder.Decode(&main); err == io.EOF {
					break
				} else if err != nil {
					return nil, err
				}
				result = append(result, main)
			}
		}
		return result, nil
	}
	b, err := goOutput(ctx, dir, goListCommand, moduleFlag, jsonFlag, allModulesArgument)
	if err != nil {
		if _, downloadErr := goOutput(ctx, dir, goModCommand, goDownloadCommand); downloadErr != nil {
			return nil, fmt.Errorf(selectedModuleDownloadFormat, downloadErr)
		}
		b, err = goOutput(ctx, dir, goListCommand, moduleFlag, jsonFlag, allModulesArgument)
		if err != nil {
			return nil, err
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(b))
	var result []module
	for {
		var m module
		if err := decoder.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// Publication contracts use the required identity, while source lookup
		// follows Go's selected replacement, including an unexpanded version.
		source := m
		if m.Replace != nil {
			source = *m.Replace
		}
		m.Dir = source.Dir
		if m.Dir == "" {
			args := []string{goModCommand, goDownloadCommand, jsonFlag, source.Path + moduleVersionSeparator + source.Version}
			data, err := goOutput(ctx, dir, args...)
			if err != nil {
				return nil, err
			}
			var downloaded module
			if err := json.Unmarshal(data, &downloaded); err != nil {
				return nil, err
			}
			m.Dir = downloaded.Dir
		}
		result = append(result, m)
	}
	return result, nil
}

func environment(ctx context.Context, dir string) (target, error) {
	b, err := goOutput(ctx, dir, goEnvCommand, jsonFlag, goOSKey, goArchKey, cgoEnabledKey, goFlagsKey)
	if err != nil {
		return target{}, err
	}
	var env target
	err = json.Unmarshal(b, &env)
	if err == nil {
		env.GOFLAGS, err = buildFlagsOnly(env.GOFLAGS)
	}
	return env, err
}

// packageLoad specifies one typed source view; overlays and build tags must travel
// together through initial analysis and generated-source validation.
type packageLoad struct {
	Dir      string
	Tags     []string
	Patterns []string
	Overlay  map[string][]byte
}

func packageConfig(ctx context.Context, request packageLoad) *packages.Config {
	cfg := packageload.Config(ctx, request.Dir, request.Overlay)
	cfg.Env = commandEnvironment(ctx)
	if len(request.Tags) > 0 {
		cfg.BuildFlags = []string{buildTagsFlagPrefix + strings.Join(request.Tags, buildTagSeparator)}
	}
	return cfg
}

func load(ctx context.Context, request packageLoad) ([]*packages.Package, error) {
	cfg := packageConfig(ctx, request)
	cfg.Mode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
		packages.NeedImports | packages.NeedDeps | packages.NeedTypes |
		packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule
	pkgs, err := packageload.Load(cfg, request.Patterns...)
	if err != nil {
		return nil, err
	}
	if err := packageload.Errors(phaseTypeChecking, pkgs); err != nil {
		return nil, err
	}
	return pkgs, nil
}

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/longxi-go/gosr/pkg/codegen/imports"

	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	manifestIndent                   = "  "
	unpublishedAnnotatedModuleFormat = "module %s contains IoC annotations but has no ioc/module.json; generate the library before publication"

	modulePublicationFormat                 = "module %s publication: %s"
	moduleVendorMetadataIsIncompleteFormat  = "module %s vendor metadata is incomplete: %w"
	moduleVendorMetadataIsUnreadableFormat  = "module %s vendor metadata is unreadable: %w"
	selectedModuleHasNoReadableSourceFormat = "selected module %s has no readable source"
)

type publication struct {
	Schema      int
	RecipeABI   int
	Module      string
	Version     string
	Target      target
	Tags        []string
	Packages    []publishedPackage
	Recipes     []publishedRecipe
	Fingerprint string
}

type publishedPackage struct {
	Path        string
	Directory   string
	Adapter     string
	AdapterHash string
	Constraint  string
	Semantics   string
	Sources     map[string]string
}

// publicationInput ties semantic metadata to the exact rendered adapters it fingerprints.
type publicationInput struct {
	Module   module
	Target   target
	Tags     []string
	Packages []*packageModel
	Recipes  []publishedRecipe
	Outputs  map[string][]byte
}

func publish(input publicationInput) ([]byte, []byte, error) {
	mod := input.Module
	manifest := publication{
		Schema:    schema,
		RecipeABI: bean.SchemaVersion,
		Module:    mod.Path,
		Version:   mod.Version,
		Target:    input.Target,
		Tags:      normalized(input.Tags),
		Recipes:   input.Recipes,
	}
	for _, m := range input.Packages {
		semantics, err := adapterSemantics(m)
		if err != nil {
			return nil, nil, err
		}
		rel, err := filepath.Rel(mod.Dir, m.Dir)
		if err != nil {
			return nil, nil, err
		}
		adapter := filepath.ToSlash(filepath.Join(rel, generatedFileName))
		manifest.Packages = append(manifest.Packages, publishedPackage{
			Path:        m.Path,
			Directory:   filepath.ToSlash(rel),
			Adapter:     adapter,
			AdapterHash: digest(input.Outputs[filepath.Join(m.Dir, generatedFileName)]),
			Semantics:   semantics,
			Sources:     m.Sources,
			Constraint:  m.Constraint,
		})
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return nil, nil, err
	}
	manifest.Fingerprint = digest(raw)
	raw, err = json.MarshalIndent(manifest, "", manifestIndent)
	if err != nil {
		return nil, nil, err
	}
	raw = append(raw, sourceLineSeparator...)
	e := newEmitter(mod.Path + importPathSeparator + indexPackageName)
	for _, p := range manifest.Packages {
		e.use(p.Path, imports.BlankAlias)
	}
	if err := e.render(indexTemplate, string(raw)); err != nil {
		return nil, nil, err
	}
	index, err := e.finish(indexPackageName, "")
	return raw, index, err
}

func normalized(values []string) []string {
	r := append([]string{}, values...)
	sort.Strings(r)
	return r
}

func readPublication(mod module, env target, tags []string) (publication, bool, error) {
	var manifest publication
	if mod.Path == runtimeModule || mod.Path == contractsModule {
		return manifest, false, nil
	}
	if mod.Dir == "" {
		return manifest, false, fmt.Errorf(selectedModuleHasNoReadableSourceFormat, mod.Path)
	}
	path := filepath.Join(mod.Dir, publicationPath)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Vendor drops JSON ancillary files, but retains the public Go constant.
		if strings.Contains(filepath.ToSlash(mod.Dir), importPathSeparator+vendorDirectory+importPathSeparator) {
			index, readErr := os.ReadFile(filepath.Join(mod.Dir, indexSourcePath))
			if readErr == nil {
				data, err = indexMetadata(index)
				if err != nil {
					return manifest, false, fmt.Errorf(moduleVendorMetadataIsIncompleteFormat, mod.Path, err)
				}
			} else if !os.IsNotExist(readErr) {
				return manifest, false, fmt.Errorf(moduleVendorMetadataIsUnreadableFormat, mod.Path, readErr)
			}
		}
	}
	if os.IsNotExist(err) {
		// Missing publication is an error only for modules declaring managed beans.
		// Annotation-free dependencies need no generated artifact.
		declared, scanErr := publicationDeclarations(mod.Dir, mod.Path, env, tags)
		found := len(declared) > 0
		if scanErr != nil {
			return manifest, false, scanErr
		}
		if found {
			return manifest, false, fmt.Errorf(unpublishedAnnotatedModuleFormat, mod.Path)
		}
		return manifest, false, nil
	}
	if err != nil {
		return manifest, false, err
	}
	validation := publicationValidation{
		module:      mod,
		target:      env,
		tags:        tags,
		data:        data,
		manifest:    &manifest,
		imports:     make(map[string]bool),
		packages:    make(map[string]bool),
		definitions: make(map[string]bool),
	}
	if err := validation.validate(); err != nil {
		return manifest, false, fmt.Errorf(modulePublicationFormat, mod.Path, err.Error())
	}
	return manifest, true, nil
}

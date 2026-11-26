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
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	duplicatePackageRecordMessage                   = "duplicate package record"
	foreignPackageInManifestMessage                 = "foreign package in manifest"
	invalidSourceFilenameMessage                    = "invalid source filename"
	invalidSourcePathMessage                        = "invalid source path"
	missingStaticAdapterSemanticsFingerprintMessage = "missing static adapter semantics fingerprint"
	publicIndexDoesNotImportAdapterPackageMessage   = "public index does not import adapter package "
	sourceFreshnessMismatchMessage                  = "source freshness mismatch: "
	staleOrMissingGeneratedAdapterMessage           = "stale or missing generated adapter "
	unrecordedSourceFileMessage                     = "unrecorded source file: "
)

func (v *publicationValidation) packageRecords() error {
	manifest, packages, imports, mod := v.manifest, v.packages, v.imports, v.module
	for _, p := range manifest.Packages {
		if p.Semantics == "" {
			return errors.New(missingStaticAdapterSemanticsFingerprintMessage)
		}
		if packages[p.Path] {
			return errors.New(duplicatePackageRecordMessage)
		}
		packages[p.Path] = true
		if !imports[p.Path] {
			return errors.New(publicIndexDoesNotImportAdapterPackageMessage + p.Path)
		}
		if p.Path != mod.Path && !strings.HasPrefix(p.Path, mod.Path+importPathSeparator) {
			return errors.New(foreignPackageInManifestMessage)
		}
		if err := v.packageSources(p); err != nil {
			return err
		}
	}

	return nil
}

func (v *publicationValidation) packageSources(p publishedPackage) error {
	mod := v.module
	for _, relative := range []string{p.Directory, p.Adapter} {
		if !safeRelative(relative) {
			return errors.New(invalidSourcePathMessage)
		}
	}
	adapter, err := os.ReadFile(filepath.Join(mod.Dir, filepath.FromSlash(p.Adapter)))
	if err != nil || digest(adapter) != p.AdapterHash {
		return errors.New(staleOrMissingGeneratedAdapterMessage + p.Adapter)
	}
	for name, hash := range p.Sources {
		if !safeRelative(name) || strings.Contains(name, importPathSeparator) {
			return errors.New(invalidSourceFilenameMessage)
		}
		source, err := os.ReadFile(filepath.Join(mod.Dir, filepath.FromSlash(p.Directory), name))
		if err != nil || digest(source) != hash {
			return errors.New(sourceFreshnessMismatchMessage + name)
		}
	}
	// The active source inventory is rechecked against the selected target by
	// packages.Load in the consumer closure. Content additions are rejected here
	// too, rather than trusting only files named by a potentially old manifest.
	entries, err := os.ReadDir(filepath.Join(mod.Dir, filepath.FromSlash(p.Directory)))
	if err != nil {
		return errors.New(err.Error())
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), goSourceExtension) && !strings.HasSuffix(entry.Name(), goTestSuffix) {
			b, err := os.ReadFile(filepath.Join(mod.Dir, filepath.FromSlash(p.Directory), entry.Name()))
			if err != nil {
				return errors.New(err.Error())
			}
			if bytes.HasPrefix(b, []byte(generatedHeader)) {
				continue
			}
			if _, ok := p.Sources[entry.Name()]; !ok {
				// Excluded variants are part of the manifest's inventory via publication
				// source scanning; an absent filename indicates stale publication.
				return errors.New(unrecordedSourceFileMessage + entry.Name())
			}
		}
	}
	return nil
}

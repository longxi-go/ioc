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
	"encoding/json"
	"errors"

	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	incompatiblePublicationMessage     = "schema, recipe ABI or module identity mismatch; regenerate the library with the matching ioctl and runtime"
	manifestFingerprintMismatchMessage = "manifest fingerprint mismatch"
	regenerateLibraryHint              = "; regenerate the library with the matching ioctl and runtime"
	versionMismatchMessage             = "version mismatch"
)

// publicationValidation checks immutable dependency metadata in diagnostic order.
// Each stage shares the same decoded manifest and established membership tables.
type publicationValidation struct {
	module      module
	target      target
	tags        []string
	data        []byte
	manifest    *publication
	imports     map[string]bool
	packages    map[string]bool
	definitions map[string]bool
}

func (v *publicationValidation) validate() error {
	if err := v.header(); err != nil {
		return err
	}
	if err := v.index(); err != nil {
		return err
	}
	if err := v.packageRecords(); err != nil {
		return err
	}
	if err := v.recipeRecords(); err != nil {
		return err
	}
	return v.closure()
}

func (v *publicationValidation) header() error {
	manifest, mod, data := v.manifest, v.module, v.data
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v.manifest); err != nil {
		return errors.New(err.Error() + regenerateLibraryHint)
	}
	if manifest.Schema != schema || manifest.RecipeABI != bean.SchemaVersion || manifest.Module != mod.Path {
		return errors.New(incompatiblePublicationMessage)
	}
	if manifest.Version != "" && manifest.Version != mod.Version {
		return errors.New(versionMismatchMessage)
	}
	fingerprint := manifest.Fingerprint
	manifest.Fingerprint = ""
	raw, err := json.Marshal(manifest)
	manifest.Fingerprint = fingerprint
	if err != nil {
		return errors.New(err.Error())
	}
	if fingerprint == "" || digest(raw) != fingerprint {
		return errors.New(manifestFingerprintMismatchMessage)
	}

	return nil
}

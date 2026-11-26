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
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	sourceannotation "github.com/longxi-go/gosr/tools/pkg/annotation"

	"sigs.k8s.io/controller-tools/pkg/markers"
)

const (
	blankValueFormat = "%s: blank value"
)

type markerTarget uint8

const (
	declarationTarget markerTarget = iota
	fieldTarget
)

type parameterMarker struct {
	Parameter  string
	Name       string   `marker:",optional"`
	Qualifiers []string `marker:",optional"`
}

type registrationMarker struct {
	Name    *string  `marker:",optional"`
	Aliases []string `marker:",optional"`
}

type injectionMarker struct {
	Name       *string  `marker:",optional"`
	Qualifiers []string `marker:",optional"`
}

type propertyMarker struct {
	Prefix         string   `marker:"prefix,optional"`
	Name           []string `marker:"name"`
	HavingValue    *string  `marker:"havingValue,optional"`
	MatchIfMissing bool     `marker:"matchIfMissing,optional"`
}

type annotation map[string][]any

func (a annotation) flag(key string) bool {
	return len(a[key]) != 0
}

func (a annotation) one(key string) string {
	if len(a[key]) == 0 {
		return ""
	}
	v, _ := a[key][0].(string)
	return v
}

func (a annotation) strings(key string) []string {
	var values []string
	for _, value := range a[key] {
		values = append(values, value.(string))
	}
	return values
}

// Registry has no function target. The explicit adapter applies declaration
// markers to FuncDecl doc groups while retaining controller-tools validation.
func readMarkers(fs *token.FileSet, group *ast.CommentGroup, target markerTarget) (annotation, error) {
	definitions := map[string]any{
		markerParam:         parameterMarker{},
		markerBean:          registrationMarker{},
		markerComponent:     registrationMarker{},
		markerScope:         "",
		markerQualifier:     "",
		markerExpose:        "",
		markerPrimary:       struct{}{},
		markerLazy:          struct{}{},
		markerOnProperty:    propertyMarker{},
		markerProfile:       []string{},
		markerOnMissingBean: conditionalMarker{},
		markerOnBean:        conditionalMarker{},
		markerOrder:         0,
		markerDependsOn:     "",
	}
	definitions[markerListener] = struct{}{}
	definitions[markerListen] = struct{}{}
	kind := markers.DescribesType
	if target == fieldTarget {
		kind = markers.DescribesField
		definitions = map[string]any{
			markerInject: injectionMarker{},
		}
	}
	schemas := make([]sourceannotation.Definition, 0, len(definitions))
	for name, sample := range definitions {
		schemas = append(schemas, sourceannotation.Definition{
			Name:       markerNamespace + name,
			Target:     kind,
			Sample:     sample,
			Repeatable: repeatableMarker(name),
		})
	}
	registry, err := sourceannotation.New(annotationPrefix, schemas)
	if err != nil {
		return nil, err
	}
	parsed, err := registry.Parse(fs, group, kind)
	if err != nil {
		return nil, err
	}
	result := annotation{}
	for _, value := range parsed {
		if text, ok := value.Value.(string); ok && strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf(blankValueFormat, value.Raw)
		}
		name := strings.TrimPrefix(value.Name, markerNamespace)
		result[name] = append(result[name], value.Value)
	}
	if err := normalizeMarkers(result); err != nil {
		return nil, err
	}
	return result, nil
}

func repeatableMarker(name string) bool {
	switch name {
	case markerParam, markerQualifier, markerExpose, markerOnProperty, markerOnMissingBean, markerOnBean, markerDependsOn:
		return true
	default:
		return false
	}
}

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
	"errors"
	"go/ast"
	"strconv"
	"strings"

	"github.com/longxi-go/ioc/internal/httpcompiler"
)

const (
	controllerExclusiveMessage        = "controller is exclusive with component and bean"
	httpBaseRequiresControllerMessage = "HTTP base requires a Controller"
	httpOneRouteMessage               = "declaration requires exactly one HTTP route"
	httpBaseMethodMessage             = "controller base route cannot declare an HTTP method"
	httpEndpointSupplementsMessage    = "HTTP endpoints cannot carry IoC registration or supplement markers"
	httpControllerScopeMessage        = "HTTP Controller scope must be singleton or request"
	httpReceiverMessage               = "HTTP receiver must be a managed Controller"
	httpLibraryMessage                = "HTTP routes are unsupported in library generation"
	httpBackendMessage                = "HTTP endpoints require an explicit HTTP backend"
	httpScalarToken                   = "+" + markerNamespace + httpcompiler.Marker + "="
	httpStructuredPathToken           = "+" + markerNamespace + httpcompiler.Marker + ":path="
	httpQuotedValuePrefix             = "\""
)

// normalizeHTTPComment quotes scalar base/shortcut paths without mutating source
// comments or relaxing controller-tools' structured attribute validation.
func normalizeHTTPComment(group *ast.CommentGroup) *ast.CommentGroup {
	if group == nil {
		return nil
	}
	scalarTokens := map[string]string{httpScalarToken: httpStructuredPathToken}
	for name := range httpcompiler.Shortcuts() {
		token := annotationPrefix + name + "="
		scalarTokens[token] = token
	}
	copyGroup := &ast.CommentGroup{}
	for _, comment := range group.List {
		copyComment := *comment
		for token, replacement := range scalarTokens {
			if before, after, ok := strings.Cut(copyComment.Text, token); ok {
				value := strings.TrimSpace(after)
				if !strings.HasPrefix(value, httpQuotedValuePrefix) {
					value = strconv.Quote(value)
				}
				copyComment.Text = before + replacement + value
				break
			}
		}
		copyGroup.List = append(copyGroup.List, &copyComment)
	}
	return copyGroup
}

func hasHTTP(a annotation) bool {
	for name := range a {
		if strings.HasPrefix(name, httpcompiler.Prefix) {
			return true
		}
	}
	return false
}

func httpMarker(a annotation) (httpcompiler.MarkerValue, error) {
	var result httpcompiler.MarkerValue
	count := 0
	if a.flag(httpcompiler.Marker) {
		result = a[httpcompiler.Marker][0].(httpcompiler.MarkerValue)
		count++
	}
	for name, method := range httpcompiler.Shortcuts() {
		if a.flag(name) {
			result = httpcompiler.MarkerValue{
				Method: method,
				Path:   a.one(name),
			}
			count++
		}
	}
	if count != 1 {
		return result, errors.New(httpOneRouteMessage)
	}
	if err := validateStaticMarkerValue(result.Path); err != nil {
		return result, err
	}
	if _, err := httpcompiler.Path("", result.Path); err != nil {
		return result, err
	}
	return result, nil
}

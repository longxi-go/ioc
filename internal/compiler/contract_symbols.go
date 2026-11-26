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
	"fmt"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

// These spellings select exported Gosr symbols in generated Go. Domain values
// are compared through the actual contract constants, never their string form.
const (
	accessTemplateFunc        = "iocAccess"
	cardinalityTemplateFunc   = "iocCardinality"
	scopeTemplateFunc         = "iocScope"
	contractSymbolPrefix      = contractsAlias + goSelector
	directSymbol              = contractSymbolPrefix + "Direct"
	deferredSymbol            = contractSymbolPrefix + "Deferred"
	oneSymbol                 = contractSymbolPrefix + "One"
	zeroOrOneSymbol           = contractSymbolPrefix + "ZeroOrOne"
	manySymbol                = contractSymbolPrefix + "Many"
	singletonSymbol           = contractSymbolPrefix + "Singleton"
	prototypeSymbol           = contractSymbolPrefix + "Prototype"
	requestSymbol             = contractSymbolPrefix + "Request"
	scopeConversionFormat     = contractSymbolPrefix + "ScopeID(%q)"
	invalidAccessMessage      = "cannot render unsupported dependency access"
	invalidCardinalityMessage = "cannot render unsupported dependency cardinality"
)

func templateAccess(value gs.Access) (string, error) {
	switch value {
	case gs.Direct:
		return directSymbol, nil
	case gs.Deferred:
		return deferredSymbol, nil
	default:
		return "", errors.New(invalidAccessMessage)
	}
}

func templateCardinality(value gs.Cardinality) (string, error) {
	switch value {
	case gs.One:
		return oneSymbol, nil
	case gs.ZeroOrOne:
		return zeroOrOneSymbol, nil
	case gs.Many:
		return manySymbol, nil
	default:
		return "", errors.New(invalidCardinalityMessage)
	}
}

func templateScope(value gs.ScopeID) string {
	switch value {
	case gs.Singleton:
		return singletonSymbol
	case gs.Prototype:
		return prototypeSymbol
	case gs.Request:
		return requestSymbol
	default:
		// Scope names are an extension point, not a closed built-in enum.
		return fmt.Sprintf(scopeConversionFormat, value)
	}
}

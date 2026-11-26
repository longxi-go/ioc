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
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

const (
	applicationListenerRequiresSingletonOwnershipMessage    = "application listener requires singleton ownership"
	contextLifecycleListenerRequiresAnEagerSingletonMessage = "context lifecycle listener requires an eager singleton"
	invalidListenerSignatureMessage                         = "listener signature must be (context.Context,E) error"
	listenMethodHasNoStaticallyIdentifiableBeanOwnerMessage = "listen method has no statically identifiable Bean owner"
	listenerEventMustImplementEventEventFormat              = "listener event %s must implement event.Event"
	listenerRequiresOnEventContextContextEErrorFormat       = "%s: listener requires OnEvent(context.Context,E) error"
	openGenericListenerReceiversAreUnsupportedMessage       = "open generic listener receivers are unsupported"
)

type annotatedMethod struct {
	object *types.Func
	order  *int
	pos    token.Pos
	used   bool
}

func attachListeners(pkg *packages.Package, model *packageModel) error {
	methods, err := annotatedListenerMethods(pkg)
	if err != nil {
		return err
	}
	for _, r := range model.Recipes {
		attachment := listenerAttachment{
			pkg:    pkg,
			recipe: r,
			seen:   make(map[string]bool),
		}
		if err := attachment.bind(methods); err != nil {
			return err
		}
	}

	for _, method := range methods {
		if !method.used {
			return at(pkg, method.pos, errors.New(listenMethodHasNoStaticallyIdentifiableBeanOwnerMessage))
		}
	}
	return nil
}

func listenerEvent(sig *types.Signature) (types.Type, error) {
	if sig.Variadic() || sig.Params().Len() != 2 ||
		!isContext(sig.Params().At(0).Type()) || sig.Results().Len() != 1 ||
		!types.Identical(sig.Results().At(0).Type(), types.Universe.Lookup(errorTypeName).Type()) {
		return nil, errors.New(invalidListenerSignatureMessage)
	}
	e := sig.Params().At(1).Type()
	marker := types.NewFunc(token.NoPos, nil, eventMarkerMethod, types.NewSignatureType(nil, nil, nil, nil, nil, false))
	boundary := types.NewInterfaceType([]*types.Func{marker}, nil).Complete()
	if !types.Implements(e, boundary) {
		return nil, fmt.Errorf(listenerEventMustImplementEventEventFormat, e)
	}
	return e, nil
}

func acceptsNotification(t types.Type) bool {
	t = types.Unalias(t)
	if contract(t) == contractsPackage+goSelector+contextRefreshedType || contract(t) == contractsPackage+goSelector+contextClosingType {
		return true
	}
	// The concrete value notifications have only EventMarker. An interface
	// accepting those values can therefore require only that one method.
	if iface, ok := t.Underlying().(*types.Interface); ok {
		iface.Complete()
		return iface.NumMethods() == 1 && iface.Method(0).Name() == eventMarkerMethod
	}
	return false
}

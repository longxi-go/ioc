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
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"golang.org/x/tools/go/packages"
)

type listenerAttachment struct {
	pkg    *packages.Package
	recipe *recipe
	seen   map[string]bool
}

func (attachment *listenerAttachment) add(selection *types.Selection, order *int, pos token.Pos) error {
	pkg, r := attachment.pkg, attachment.recipe
	method := selection.Obj().(*types.Func)
	e, err := listenerEvent(method.Type().(*types.Signature))
	if err != nil {
		return at(pkg, pos, err)
	}
	if attachment.seen[method.Name()] {
		for i := range r.Listeners {
			if r.Listeners[i].Method == method.Name() && order != nil {
				r.Listeners[i].Order = order
			}
		}
		return nil
	}
	attachment.seen[method.Name()] = true
	lifecycle := acceptsNotification(e)
	if r.Definition.Scope != gs.Singleton {
		return at(pkg, pos, errors.New(applicationListenerRequiresSingletonOwnershipMessage))
	}
	if lifecycle && r.Definition.Lazy {
		return at(pkg, pos, errors.New(contextLifecycleListenerRequiresAnEagerSingletonMessage))
	}
	r.Listeners = append(r.Listeners, listener{
		Method:    method.Name(),
		ID:        r.Definition.ID + definitionMemberSeparator + method.Name(),
		Event:     e,
		Source:    source(pkg, pos),
		Order:     order,
		Lifecycle: lifecycle,
	})
	return nil
}

func (attachment *listenerAttachment) bind(methods []*annotatedMethod) error {
	pkg, r := attachment.pkg, attachment.recipe

	methodSet := types.NewMethodSet(r.Result)
	if r.ListenerType {
		selection := methodSet.Lookup(nil, onEventMethod)
		if selection == nil {
			return fmt.Errorf(listenerRequiresOnEventContextContextEErrorFormat, r.Definition.ID)
		}
		if err := attachment.add(selection, nil, selection.Obj().Pos()); err != nil {
			return err
		}
	}
	// Interface-result factories do not identify concrete implementation
	// annotations; only the statically known concrete result may own them.
	if _, ok := types.Unalias(r.Result).Underlying().(*types.Interface); ok {
		return nil
	}
	for _, declaration := range methods {
		selection := methodSet.Lookup(pkg.Types, declaration.object.Name())
		if selection == nil || selection.Obj() != declaration.object {
			continue
		}
		declaration.used = true
		if err := attachment.add(selection, declaration.order, declaration.pos); err != nil {
			return err
		}
	}
	slices.SortFunc(r.Listeners, func(a, b listener) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return nil
}

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

package eventdispatch

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"

	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	invalidOrDuplicateListenerDeclarationMessage = "invalid or duplicate listener declaration "
	listenerInstanceUnavailableMessage           = "listener instance unavailable"
	listenerInvocationFailedMessage              = "listener invocation failed"
	missingInstanceResolverMessage               = "missing instance resolver"
	// NilEventMessage identifies an absent event at both publication boundaries.
	NilEventMessage = "nil event interface"
)

// Entry ties one static declaration to its instance-owning definition.
type Entry struct {
	Bean     string
	Listener bean.Listener
}

// Access keeps resource ownership and phase gates in the container. Resolve's
// final argument forbids construction when true; false found means unavailable.
// Check is called before preparation, invocation and a successful return.
type Access struct {
	Resolve listenerResolveFunc
	Check   contextCheckFunc
}

// Dispatcher holds copied descriptors, never listener instances or caller contexts.
type Dispatcher struct {
	entries []Entry
	mu      sync.Mutex
	active  map[*frame]bool
}

// New validates structural declarations without invoking their adapters.
func New(entries []Entry) (*Dispatcher, error) {
	copied := make([]Entry, len(entries))
	seen := map[string]bool{}
	for i, entry := range entries {
		l := entry.Listener
		if strings.TrimSpace(entry.Bean) == "" ||
			strings.TrimSpace(l.ID) == "" ||
			strings.TrimSpace(l.Event) == "" ||
			l.Source.Line < 0 ||
			l.Match == nil ||
			l.Invoke == nil ||
			seen[l.ID] {
			return nil, &gs.Error{
				Code:    gs.InvalidDefinition,
				Bean:    entry.Bean,
				Source:  l.Source,
				Message: invalidOrDuplicateListenerDeclarationMessage + l.ID,
			}
		}
		seen[l.ID] = true
		if l.Order != nil {
			order := *l.Order
			l.Order = &order
		}
		copied[i] = Entry{
			Bean:     entry.Bean,
			Listener: l,
		}
	}
	slices.SortFunc(copied, func(a, b Entry) int {
		return strings.Compare(a.Listener.ID, b.Listener.ID)
	})
	return &Dispatcher{
		entries: copied,
		active:  map[*frame]bool{},
	}, nil
}

// Deliver is synchronous and fail-fast. All matching instances and ordering
// keys prepare successfully before any listener callback starts.
func (d *Dispatcher) Deliver(ctx context.Context, value event.Event, access Access) error {
	return d.deliver(ctx, value, access, false)
}

// DeliverExisting is a best-effort cleanup notification. It never requests a
// new instance and joins preparation/order/callback errors after other attempts.
func (d *Dispatcher) DeliverExisting(ctx context.Context, value event.Event, access Access) error {
	return d.deliver(ctx, value, access, true)
}

type receiver struct {
	value   any
	key     orderKey
	err     error
	sampled bool
}

type invocation struct {
	entry Entry
	value any
	key   orderKey
}

func (d *Dispatcher) deliver(ctx context.Context, value event.Event, access Access, existing bool) error {
	if err := check(ctx, access); err != nil {
		return err
	}
	if value == nil {
		return &event.Error{
			Code:    event.InvalidEvent,
			Message: NilEventMessage,
		}
	}
	calls, failures, err := d.prepare(ctx, value, access, existing)
	if err != nil {
		return errors.Join(append(failures, err)...)
	}
	sortInvocations(calls)
	for _, call := range calls {
		if err := check(ctx, access); err != nil {
			return errors.Join(append(failures, err)...)
		}
		err := d.invoke(ctx, call, value)
		if err != nil {
			failure := listenerFailure(call.entry, listenerInvocationFailedMessage, err)
			if !existing {
				return failure
			}
			failures = append(failures, failure)
		}
	}
	if err := check(ctx, access); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

func prepare(ctx context.Context, access Access, id string, existing bool) receiver {
	var value any
	var found bool
	err := protect(func() error {
		if access.Resolve == nil {
			return &gs.Error{
				Code:    gs.InvalidDefinition,
				Bean:    id,
				Message: missingInstanceResolverMessage,
			}
		}
		var err error
		value, found, err = access.Resolve(ctx, id, existing)
		return err
	})
	if err == nil && (!found || value == nil) {
		err = &gs.Error{
			Code:    gs.MissingBean,
			Bean:    id,
			Message: listenerInstanceUnavailableMessage,
		}
	}
	return receiver{
		value: value,
		err:   err,
	}
}

func check(ctx context.Context, access Access) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if access.Check != nil {
		return access.Check(ctx)
	}
	return nil
}

func listenerFailure(entry Entry, message string, cause error) error {
	return &event.Error{
		Code:     event.ListenerFailed,
		Listener: entry.Listener.ID,
		Cause: &gs.Error{
			Code:    gs.CreationFailed,
			Bean:    entry.Bean,
			Source:  entry.Listener.Source,
			Message: message,
			Cause:   cause,
		},
	}
}

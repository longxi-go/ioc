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

	"github.com/longxi-go/gosr/pkg/event"
)

const (
	eventMatchingFailedMessage       = "event matching failed"
	listenerOrderingFailedMessage    = "listener ordering failed"
	listenerPreparationFailedMessage = "listener preparation failed"
)

// Separate phases make the preparation boundary observable: order methods can
// rely on every matching owner having finished initialization, while no business
// listener runs until all instances and keys are ready.
func (d *Dispatcher) prepare(ctx context.Context, value event.Event, access Access, existing bool) ([]invocation, []error, error) {
	matched, failures, err := d.match(ctx, value, access, existing)
	if err != nil {
		return nil, failures, err
	}
	owners := map[string]receiver{}
	for _, entry := range matched {
		if err := check(ctx, access); err != nil {
			return nil, failures, err
		}
		if _, resolved := owners[entry.Bean]; resolved {
			continue
		}
		object := prepare(ctx, access, entry.Bean, existing)
		owners[entry.Bean] = object
		if object.err != nil {
			failure := listenerFailure(entry, listenerPreparationFailedMessage, object.err)
			if !existing {
				return nil, failures, failure
			}
			failures = append(failures, failure)
		}
	}
	var calls []invocation
	for _, entry := range matched {
		if err := check(ctx, access); err != nil {
			return nil, failures, err
		}
		object := owners[entry.Bean]
		if object.err != nil {
			continue
		}
		if !object.sampled {
			object.key, err = sample(object.value)
			object.sampled = true
			owners[entry.Bean] = object
			if err != nil {
				failure := listenerFailure(entry, listenerOrderingFailedMessage, err)
				if !existing {
					return nil, failures, failure
				}
				failures = append(failures, failure)
			}
		}
		key := object.key
		if entry.Listener.Order != nil {
			key.order = *entry.Listener.Order
		}
		calls = append(calls, invocation{
			entry: entry,
			value: object.value,
			key:   key,
		})
	}
	return calls, failures, nil
}

func (d *Dispatcher) match(ctx context.Context, value event.Event, access Access, existing bool) ([]Entry, []error, error) {
	var matched []Entry
	var failures []error
	for _, entry := range d.entries {
		if err := check(ctx, access); err != nil {
			return nil, failures, err
		}
		accepts := false
		err := protect(func() error {
			accepts = entry.Listener.Match(value)
			return nil
		})
		if err != nil {
			failure := listenerFailure(entry, eventMatchingFailedMessage, err)
			if !existing {
				return nil, failures, failure
			}
			failures = append(failures, failure)
		} else if accepts {
			matched = append(matched, entry)
		}
	}
	return matched, failures, nil
}

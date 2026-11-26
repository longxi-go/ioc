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
	"fmt"
	"slices"

	"github.com/longxi-go/gosr/pkg/event"
)

const (
	listenerPanicCauseFormat = "listener panic: %w"
	listenerPanicValueFormat = "listener panic: %v"
)

type (
	frameKey struct{}

	frame struct {
		dispatcher *Dispatcher
		id         string
		previous   *frame
	}
)

func (d *Dispatcher) invoke(ctx context.Context, call invocation, value event.Event) error {
	callCtx, finish, err := d.enter(ctx, call.entry.Listener.ID)
	if err != nil {
		return err
	}
	defer finish()
	return protect(func() error {
		return call.entry.Listener.Invoke(callCtx, call.value, value)
	})
}

func (d *Dispatcher) enter(ctx context.Context, id string) (context.Context, completionFunc, error) {
	previous, _ := ctx.Value(frameKey{}).(*frame)
	d.mu.Lock()
	var path []string
	for current := previous; current != nil; current = current.previous {
		if current.dispatcher == d && d.active[current] {
			path = append(path, current.id)
		}
	}
	if slices.Contains(path, id) {
		slices.Reverse(path)
		d.mu.Unlock()
		return nil, nil, &event.Error{
			Code:     event.PublicationCycle,
			Listener: id,
			Path:     append(path, id),
		}
	}
	current := &frame{
		dispatcher: d,
		id:         id,
		previous:   previous,
	}
	d.active[current] = true
	d.mu.Unlock()
	finish := func() {
		d.mu.Lock()
		delete(d.active, current)
		d.mu.Unlock()
	}
	return context.WithValue(ctx, frameKey{}, current), finish, nil
}

func protect(callback protectedCallFunc) (err error) {
	defer func() {
		if value := recover(); value != nil {
			if cause, ok := value.(error); ok {
				err = fmt.Errorf(listenerPanicCauseFormat, cause)
			} else {
				err = fmt.Errorf(listenerPanicValueFormat, value)
			}
		}
	}()
	return callback()
}

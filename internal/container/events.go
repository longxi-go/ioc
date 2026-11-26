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

package container

import (
	"context"
	"errors"
	"slices"

	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/internal/eventdispatch"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	lifecycleListenerOwnershipMessage      = "context lifecycle listeners require eager singleton ownership"
	listenerLifecycleMatchingFailedMessage = "listener lifecycle matching failed"

	publicationPhaseEndedMessage            = "publication phase ended"
	publicationRequiresARunningOwnerMessage = "publication requires a running owner"
	reservedLifecycleNotificationMessage    = "context notifications are reserved for their owning lifecycle"
	reservedPublisherPrimaryMessage         = "the local event publisher is already primary; select alternatives by name"
)

// publisher exposes only publication, so injecting the narrow capability cannot
// accidentally expose the Engine's lookup or lifecycle methods.
type publisher struct {
	engine *Engine
}

func (p *publisher) PublishEvent(ctx context.Context, value event.Event) error {
	return p.engine.PublishEvent(ctx, value)
}

func (e *Engine) publisherRecipe() bean.Registration {
	return bean.Registration{
		Definition: gs.BeanDefinition{
			ID:        bean.EventPublisherID,
			Name:      bean.EventPublisherName,
			Contracts: []gs.Contract{bean.EventPublisherContract},
			Scope:     gs.Singleton,
			Primary:   true,
		},
		Create: func(context.Context, bean.Ports) (any, error) {
			return &publisher{
				engine: e,
			}, nil
		},
		Accept: func(value any) bool {
			_, ok := value.(event.Publisher)
			return ok
		},
	}
}

func newEvents(bs []*bound) (*eventdispatch.Dispatcher, error) {
	var entries []eventdispatch.Entry
	for _, b := range bs {
		r := b.recipe
		if r.Definition.ID != bean.EventPublisherID &&
			r.Definition.Primary &&
			slices.Contains(r.Definition.Contracts, bean.EventPublisherContract) {
			return nil, recipeFailure(r, gs.InvalidDefinition, reservedPublisherPrimaryMessage, nil)
		}
		for _, listener := range r.Listeners {
			lifecycle := false
			err := invoke(func() error {
				lifecycle = listener.Match(gs.ContextRefreshed{}) || listener.Match(gs.ContextClosing{})
				return nil
			})
			if err != nil {
				return nil, recipeFailure(r, gs.InvalidDefinition, listenerLifecycleMatchingFailedMessage, err)
			}
			if lifecycle && r.Definition.Lazy {
				return nil, recipeFailure(r, gs.InvalidDefinition, lifecycleListenerOwnershipMessage, nil)
			}
			entries = append(entries, eventdispatch.Entry{
				Bean:     r.Definition.ID,
				Listener: listener,
			})
		}
	}
	return eventdispatch.New(entries)
}

// PublishEvent owns an entire synchronous publication, including all callbacks.
// New work is admitted only while Running; metadata matching never constructs
// unrelated lazy listeners and scope capabilities cannot outlive the call.
func (e *Engine) PublishEvent(ctx context.Context, value event.Event) error {
	e.mu.Lock()
	running := e.state == gs.Running
	e.mu.Unlock()
	if !running {
		return &event.Error{
			Code:    event.InvalidState,
			Message: publicationRequiresARunningOwnerMessage,
		}
	}
	call, t, finish, err := e.begin(ctx)
	if err != nil {
		if errors.Is(err, &gs.Error{
			Code: gs.InvalidState,
		}) {
			return &event.Error{
				Code:  event.InvalidState,
				Cause: err,
			}
		}
		return err
	}
	defer finish()
	if value == nil {
		return &event.Error{
			Code:    event.InvalidEvent,
			Message: eventdispatch.NilEventMessage,
		}
	}
	switch value.(type) {
	case gs.ContextRefreshed, *gs.ContextRefreshed, gs.ContextClosing, *gs.ContextClosing:
		return &event.Error{
			Code:    event.InvalidEvent,
			Message: reservedLifecycleNotificationMessage,
		}
	}
	call, release, err := e.pinPublication(call, t)
	if err != nil {
		return err
	}
	defer release()
	return e.events.Deliver(call, value, e.eventAccess(gs.Running))
}

func (e *Engine) eventAccess(phase gs.State) eventdispatch.Access {
	return eventdispatch.Access{
		Check: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			e.mu.Lock()
			valid := e.state == phase
			e.mu.Unlock()
			if !valid {
				return &event.Error{
					Code:    event.InvalidState,
					Message: publicationPhaseEndedMessage,
				}
			}
			return nil
		},
		Resolve: e.eventReceiver,
	}
}

func (e *Engine) eventReceiver(ctx context.Context, id string, existing bool) (any, bool, error) {
	var target *bound
	for _, b := range e.bound {
		if b.recipe.Definition.ID == id {
			target = b
			break
		}
	}
	if target == nil {
		return nil, false, nil
	}
	if existing {
		e.mu.Lock()
		s := target.singleton
		value, ready := s.value, s.done != nil && s.task == nil && s.err == nil
		e.mu.Unlock()
		return value, ready, nil
	}
	call, child, finish, err := e.begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer finish()
	value, err := e.get(call, target, resolution{
		task: child,
	})
	return value, err == nil, err
}

func (e *Engine) refreshed(ctx context.Context, t *task) error {
	call, release, err := e.pinPublication(ctx, t)
	if err != nil {
		return err
	}
	defer release()
	return e.events.Deliver(call, gs.ContextRefreshed{}, e.eventAccess(gs.Starting))
}

func (e *Engine) closingEvent() error {
	e.mu.Lock()
	if !e.everRunning {
		e.mu.Unlock()
		return nil
	}
	t := e.addTask(nil, true, nil)
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.removeTask(t)
		e.mu.Unlock()
	}()
	ctx := context.WithValue(context.Background(), taskKey{}, taskToken{
		e,
		t,
	})
	return e.events.DeliverExisting(ctx, gs.ContextClosing{}, e.eventAccess(gs.Closing))
}

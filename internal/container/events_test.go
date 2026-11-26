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
	"testing"

	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

type (
	businessEvent struct {
		event.BaseEvent
	}

	unrelatedEvent struct {
		event.BaseEvent
	}
)

type eventCallbackFunc[E event.Event] = func(context.Context, E) error

func listening[E event.Event](id, method string, callback eventCallbackFunc[E]) bean.Registration {
	r := constant(id)
	r.Listeners = []bean.Listener{
		{
			ID:    id + "." + method,
			Event: "test.Event",
			Match: func(value event.Event) bool {
				_, ok := value.(E)
				return ok
			},
			Invoke: func(ctx context.Context, _ any, value event.Event) error {
				return callback(ctx, value.(E))
			},
		},
	}
	return r
}

func TestPublicationGatesPrecedePayloadValidation(t *testing.T) {
	ctx := context.Background()
	e, err := New(nil, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.PublishEvent(ctx, nil); !errors.Is(err, &event.Error{
		Code: event.InvalidState,
	}) {
		t.Fatal("created publication", err)
	}
	if err := e.Start(ctx); err != nil {
		t.Fatal(err)
	}
	for _, value := range []event.Event{
		nil,
		gs.ContextRefreshed{},
		&gs.ContextRefreshed{},
		gs.ContextClosing{},
		(*gs.ContextClosing)(nil),
	} {
		if err := e.PublishEvent(ctx, value); !errors.Is(err, &event.Error{
			Code: event.InvalidEvent,
		}) {
			t.Fatalf("reserved/nil publication %T: %v", value, err)
		}
	}
	if err := e.PublishEvent(ctx, businessEvent{}); err != nil {
		t.Fatal("empty admitted publication", err)
	}
	if err := e.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e.PublishEvent(ctx, nil); !errors.Is(err, &event.Error{
		Code: event.InvalidState,
	}) {
		t.Fatal("closed publication", err)
	}
}

func TestPublicationOnlyCreatesMatchingLazyListener(t *testing.T) {
	created, calls := 0, 0
	r := listening("lazy", "OnEvent", func(context.Context, businessEvent) error {
		calls++
		return nil
	})
	r.Definition.Lazy = true
	r.Create = func(context.Context, bean.Ports) (any, error) {
		created++
		return new(int), nil
	}
	e := boot(t, r)
	if created != 0 {
		t.Fatal("startup created business-only lazy listener")
	}
	if err := e.PublishEvent(context.Background(), unrelatedEvent{}); err != nil || created != 0 {
		t.Fatal("unrelated publication created listener", created, err)
	}
	for range 2 {
		if err := e.PublishEvent(context.Background(), businessEvent{}); err != nil {
			t.Fatal(err)
		}
	}
	if created != 1 || calls != 2 {
		t.Fatal(created, calls)
	}
}

func TestContextEventsFollowManagedLifecycleBoundaries(t *testing.T) {
	ctx := context.Background()
	var sequence []string
	var e *Engine
	r := listening("listener", "OnRefreshed", func(call context.Context, _ gs.ContextRefreshed) error {
		sequence = append(sequence, "refreshed")
		if e.State() != gs.Starting {
			t.Error("startup barrier observed committed Running")
		}
		if _, err := e.Lookup(call, gs.Selector{
			Name: "resource",
		}); err != nil {
			t.Error("internal startup lookup", err)
		}
		if err := e.PublishEvent(call, businessEvent{}); !errors.Is(err, &event.Error{
			Code: event.InvalidState,
		}) {
			t.Error("startup business publication admitted", err)
		}
		return nil
	})
	closing := listening("listener", "OnClosing", func(call context.Context, _ gs.ContextClosing) error {
		sequence = append(sequence, "closing")
		if _, err := e.Lookup(call, gs.Selector{
			Name: "resource",
		}); !errors.Is(err, &gs.Error{
			Code: gs.InvalidState,
		}) {
			t.Error("closing lookup admitted", err)
		}
		if err := e.Close(call); !errors.Is(err, &gs.Error{
			Code: gs.InvalidState,
		}) {
			t.Error("closing callback joined itself", err)
		}
		return nil
	})
	r.Listeners = append(r.Listeners, closing.Listeners...)
	r.Init = func(context.Context, any) error {
		sequence = append(sequence, "init")
		return nil
	}
	r.Start = func(context.Context, any) error {
		sequence = append(sequence, "start")
		return nil
	}
	r.Stop = func(context.Context, any) error {
		sequence = append(sequence, "stop")
		return nil
	}
	r.Destroy = func(context.Context, any) error {
		sequence = append(sequence, "destroy")
		return nil
	}
	var err error
	e, err = New([]bean.Registration{r, constant("resource")}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Start(ctx); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := e.Close(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(sequence, []string{"init", "start", "refreshed", "closing", "stop", "destroy"}) {
		t.Fatal(sequence)
	}
}

func TestRefreshedFailureCompensatesAndDoesNotSendClosing(t *testing.T) {
	cause := errors.New("startup observer")
	stopped, destroyed, closing := 0, 0, 0
	r := listening("listener", "OnRefreshed", func(context.Context, gs.ContextRefreshed) error {
		return cause
	})
	c := listening("listener", "OnClosing", func(context.Context, gs.ContextClosing) error {
		closing++
		return nil
	})
	r.Listeners = append(r.Listeners, c.Listeners...)
	r.Stop = func(context.Context, any) error {
		stopped++
		return nil
	}
	r.Destroy = func(context.Context, any) error {
		destroyed++
		return nil
	}
	e, err := New([]bean.Registration{r}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Start(context.Background()); !errors.Is(err, cause) ||
		e.State() != gs.Failed ||
		stopped != 1 ||
		destroyed != 1 {
		t.Fatalf("startup compensation: stop=%d destroy=%d state=%s error=%v", stopped, destroyed, e.State(), err)
	}
	if err := e.Close(context.Background()); err != nil || closing != 0 || destroyed != 1 {
		t.Fatal("failed context sent Closing or cleaned twice", closing, destroyed, err)
	}
}

func TestLifecycleListenerValidationPrecedesFactories(t *testing.T) {
	for _, kind := range []string{"event-wide lazy", "request", "duplicate", "alternative primary"} {
		t.Run(kind, func(t *testing.T) {
			created := 0
			r := listening("listener", "OnEvent", func(context.Context, event.Event) error {
				return nil
			})
			r.Create = func(context.Context, bean.Ports) (any, error) {
				created++
				return new(int), nil
			}
			switch kind {
			case "event-wide lazy":
				r.Definition.Lazy = true
			case "request":
				r.Definition.Scope = gs.Request
			case "duplicate":
				r.Listeners = append(r.Listeners, r.Listeners[0])
			case "alternative primary":
				r.Definition.Contracts = []gs.Contract{bean.EventPublisherContract}
				r.Definition.Primary = true
			}
			e, err := New([]bean.Registration{r}, Options{
				Snapshot: conditionSnapshot(t, nil),
			})
			if err == nil {
				err = e.Start(context.Background())
				if closeErr := e.Close(context.Background()); closeErr != nil {
					t.Error(closeErr)
				}
			}
			if !errors.Is(err, &gs.Error{
				Code: gs.InvalidDefinition,
			}) || created != 0 {
				t.Fatal("invalid listener reached allocation", created, err)
			}
		})
	}
}

func TestClosingAggregatesFailuresAndStillDestroys(t *testing.T) {
	first, second, cleanup := errors.New("first observer"), errors.New("second observer"), errors.New("destroy")
	calls, destroyed := 0, 0
	a := listening("a", "OnClosing", func(context.Context, gs.ContextClosing) error {
		calls++
		return first
	})
	b := listening("b", "OnClosing", func(context.Context, gs.ContextClosing) error {
		calls++
		return second
	})
	a.Destroy = func(context.Context, any) error {
		destroyed++
		return cleanup
	}
	e, err := New([]bean.Registration{a, b}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		err := e.Close(context.Background())
		if !errors.Is(err, first) ||
			!errors.Is(err, second) ||
			!errors.Is(err, cleanup) ||
			calls != 2 ||
			destroyed != 1 ||
			e.State() != gs.Closed {
			t.Fatal("closing lost causes or abandoned/duplicated cleanup", calls, destroyed, err)
		}
	}
}

func TestRefreshedFailureOwnsResourcesCreatedInsideBarrier(t *testing.T) {
	cause := errors.New("barrier failure")
	created, destroyed := 0, 0
	late := constant("late")
	late.Definition.Lazy = true
	late.Create = func(context.Context, bean.Ports) (any, error) {
		created++
		return new(int), nil
	}
	late.Destroy = func(context.Context, any) error {
		destroyed++
		return nil
	}
	var e *Engine
	r := listening("listener", "OnRefreshed", func(ctx context.Context, _ gs.ContextRefreshed) error {
		if _, err := e.Lookup(ctx, gs.Selector{
			Name: "late",
		}); err != nil {
			return err
		}
		return cause
	})
	var err error
	e, err = New([]bean.Registration{r, late}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Start(context.Background()); !errors.Is(err, cause) || created != 1 || destroyed != 1 {
		t.Fatal("late allocation escaped startup compensation", created, destroyed, err)
	}
	if err := e.Close(context.Background()); err != nil || destroyed != 1 {
		t.Fatal(destroyed, err)
	}
}

func TestConcurrentCloseDuringRefreshedCannotCommitRunning(t *testing.T) {
	entered, canceled := make(chan struct{}), make(chan struct{})
	closing := 0
	r := listening("listener", "OnRefreshed", func(ctx context.Context, _ gs.ContextRefreshed) error {
		close(entered)
		<-ctx.Done()
		close(canceled)
		return nil
	})
	c := listening("listener", "OnClosing", func(context.Context, gs.ContextClosing) error {
		closing++
		return nil
	})
	r.Listeners = append(r.Listeners, c.Listeners...)
	e, err := New([]bean.Registration{r}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	started, closed := make(chan error, 1), make(chan error, 1)
	go func() {
		started <- e.Start(context.Background())
	}()
	<-entered
	go func() {
		closed <- e.Close(context.Background())
	}()
	<-canceled
	if err := <-started; err == nil {
		t.Fatal("canceled startup committed Running")
	}
	if err := <-closed; err != nil || e.State() != gs.Closed || closing != 0 {
		t.Fatal("failed startup sent normal Closing", closing, err)
	}
}

func TestInactiveListenersDoNotCreateOrReceiveEvents(t *testing.T) {
	r := listening("inactive", "OnEvent", func(context.Context, event.Event) error {
		t.Error("inactive listener invoked")
		return nil
	})
	r.Create = func(context.Context, bean.Ports) (any, error) {
		t.Error("inactive factory invoked")
		return new(int), nil
	}
	r.Conditions = []bean.Condition{
		{
			Property: &bean.PropertyCondition{
				Names:       []string{"enabled"},
				HavingValue: new("true"),
			},
		},
	}
	e, err := New([]bean.Registration{r}, Options{
		Snapshot: conditionSnapshot(t, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	if err := e.PublishEvent(context.Background(), businessEvent{}); err != nil {
		t.Fatal(err)
	}
}

func TestCanceledCloseWaitDoesNotAbandonClosingNotification(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	destroyed := 0
	r := listening("listener", "OnClosing", func(context.Context, gs.ContextClosing) error {
		close(entered)
		<-release
		return nil
	})
	r.Destroy = func(context.Context, any) error {
		destroyed++
		return nil
	}
	e := boot(t, r)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.Close(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled waiter did not return", err)
	}
	<-entered
	close(release)
	if err := e.Close(context.Background()); err != nil || destroyed != 1 || e.State() != gs.Closed {
		t.Fatal("cleanup ownership abandoned", destroyed, err)
	}
}

func TestLazyListenerPreparationPanicPreservesCauseBeforeCallbacks(t *testing.T) {
	for _, phase := range []string{"factory", "decorator", "accept", "init"} {
		t.Run(phase, func(t *testing.T) {
			cause := errors.New("preparation panic cause")
			callbacks := 0
			good := listening("a.ready", "OnEvent", func(context.Context, businessEvent) error {
				callbacks++
				return nil
			})
			bad := listening("z.bad", "OnEvent", func(context.Context, businessEvent) error {
				callbacks++
				return nil
			})
			good.Definition.Lazy, bad.Definition.Lazy = true, true
			opts := Options{
				Snapshot: conditionSnapshot(t, nil),
			}
			switch phase {
			case "factory":
				bad.Create = func(context.Context, bean.Ports) (any, error) {
					panic(cause)
				}
			case "decorator":
				opts.Decorators = []bean.Decorator{
					{
						Contract: "z.bad",
						Wrap: func(context.Context, any) (any, error) {
							panic(cause)
						},
					},
				}
			case "accept":
				bad.Accept = func(any) bool {
					panic(cause)
				}
			case "init":
				bad.Init = func(context.Context, any) error {
					panic(cause)
				}
			}
			e, err := New([]bean.Registration{good, bad}, opts)
			if err != nil {
				t.Fatal(err)
			}
			if err := e.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			err = e.PublishEvent(context.Background(), businessEvent{})
			if !errors.Is(err, cause) || !errors.Is(err, &event.Error{
				Code: event.ListenerFailed,
			}) || callbacks != 0 {
				t.Fatalf("preparation cause or barrier lost: callbacks=%d error=%v", callbacks, err)
			}
			if err := e.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

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

package eventdispatch_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/longxi-go/gosr/pkg/event"
	"github.com/longxi-go/gosr/pkg/ordered"
	"github.com/longxi-go/ioc/internal/eventdispatch"
	"github.com/longxi-go/ioc/pkg/bean"
)

type note struct {
	event.BaseEvent
}

func entry(owner, id string, match bean.EventMatchFunc, invoke bean.EventInvokeFunc) eventdispatch.Entry {
	return eventdispatch.Entry{
		Bean: owner,
		Listener: bean.Listener{
			ID:     id,
			Event:  "note",
			Match:  match,
			Invoke: invoke,
		},
	}
}

type preparationObserver struct {
	ready *bool
}

func (o preparationObserver) Order() int {
	if !*o.ready {
		panic("order sampled before matching instances finished preparation")
	}
	return 0
}

func TestAllMatchingInstancesResolveBeforeOrderSampling(t *testing.T) {
	ready := false
	calls := 0
	run := func(context.Context, any, event.Event) error {
		calls++
		return nil
	}
	d, err := eventdispatch.New([]eventdispatch.Entry{entry("first", "a", matchesNote, run), entry("second", "b", matchesNote, run)})
	if err != nil {
		t.Fatal(err)
	}
	a := access(nil)
	a.Resolve = func(_ context.Context, id string, _ bool) (any, bool, error) {
		if id == "first" {
			return preparationObserver{
				ready: &ready,
			}, true, nil
		}
		ready = true
		return struct{}{}, true, nil
	}
	if err := d.Deliver(context.Background(), note{}, a); err != nil || calls != 2 {
		t.Fatalf("preparation boundary: calls=%d error=%v", calls, err)
	}
}

type followup struct {
	event.BaseEvent
}

func TestNestedMethodsUseDeclarationIdentityAndRejectActiveReentry(t *testing.T) {
	var d *eventdispatch.Dispatcher
	var a eventdispatch.Access
	reenter := false
	depth := 0
	var calls []string
	entries := []eventdispatch.Entry{
		entry("same-owner", "owner.A", matchesNote, func(ctx context.Context, _ any, _ event.Event) error {
			depth++
			defer func() {
				depth--
			}()
			if depth > 2 {
				return errors.New("cycle was not detected")
			}
			calls = append(calls, "A")
			return d.Deliver(ctx, followup{}, a)
		}),
		entry("same-owner", "owner.B", func(e event.Event) bool {
			_, ok := e.(followup)
			return ok
		}, func(ctx context.Context, _ any, _ event.Event) error {
			calls = append(calls, "B")
			if reenter {
				return d.Deliver(ctx, note{}, a)
			}
			return nil
		}),
	}
	var err error
	d, err = eventdispatch.New(entries)
	if err != nil {
		t.Fatal(err)
	}
	a = access(map[string]any{
		"same-owner": struct{}{},
	})
	if err := d.Deliver(context.Background(), note{}, a); err != nil || !slices.Equal(calls, []string{"A", "B"}) {
		t.Fatal("distinct methods must nest", calls, err)
	}
	reenter = true
	err = d.Deliver(context.Background(), note{}, a)
	if !errors.Is(err, &event.Error{
		Code: event.PublicationCycle,
	}) || !strings.Contains(err.Error(), "owner.A -> owner.B -> owner.A") {
		t.Fatalf("missing declaration cycle/path: %v", err)
	}
}

func TestCompletedFramesAndIndependentConcurrentCallsDoNotShareVisitedState(t *testing.T) {
	var saved context.Context
	var count atomic.Int64
	var capture atomic.Bool
	capture.Store(true)
	d, err := eventdispatch.New([]eventdispatch.Entry{entry("owner", "owner.A", matchesNote, func(ctx context.Context, _ any, _ event.Event) error {
		if capture.Load() {
			saved = ctx
		}
		count.Add(1)
		return nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	a := access(map[string]any{
		"owner": struct{}{},
	})
	if err := d.Deliver(context.Background(), note{}, a); err != nil {
		t.Fatal(err)
	}
	capture.Store(false)
	if err := d.Deliver(saved, note{}, a); err != nil {
		t.Fatal("completed frame treated as active", err)
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if err := d.Deliver(context.Background(), note{}, a); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if count.Load() != 22 {
		t.Fatal("independent calls were suppressed", count.Load())
	}
}

func matchesNote(e event.Event) bool {
	_, ok := e.(note)
	return ok
}

func access(objects map[string]any) eventdispatch.Access {
	return eventdispatch.Access{
		Resolve: func(_ context.Context, id string, _ bool) (any, bool, error) {
			value, ok := objects[id]
			return value, ok, nil
		},
		Check: func(ctx context.Context) error {
			return ctx.Err()
		},
	}
}

func TestMatchingPreservesValuePointerAndInterfaceSemantics(t *testing.T) {
	var calls []string
	record := func(id string) bean.EventInvokeFunc {
		return func(context.Context, any, event.Event) error {
			calls = append(calls, id)
			return nil
		}
	}
	d, err := eventdispatch.New([]eventdispatch.Entry{
		entry("receiver", "a.value", matchesNote, record("value")),
		entry("receiver", "b.pointer", func(e event.Event) bool {
			_, ok := e.(*note)
			return ok
		}, record("pointer")),
		entry("receiver", "z.event", func(event.Event) bool {
			return true
		}, record("event")),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		value event.Event
		want  []string
	}{
		{
			note{},
			[]string{"value", "event"},
		},
		{
			&note{},
			[]string{"pointer", "event"},
		},
		{
			(*note)(nil),
			[]string{"pointer", "event"},
		},
	} {
		calls = nil
		if err := d.Deliver(context.Background(), tc.value, access(map[string]any{
			"receiver": struct{}{},
		})); err != nil || !slices.Equal(calls, tc.want) {
			t.Fatalf("delivery %T: calls=%v error=%v", tc.value, calls, err)
		}
	}
}

func TestPreparationFailureStartsNoCallbacks(t *testing.T) {
	cause := errors.New("late receiver construction")
	calls := 0
	run := func(context.Context, any, event.Event) error {
		calls++
		return nil
	}
	d, err := eventdispatch.New([]eventdispatch.Entry{entry("first", "a", matchesNote, run), entry("second", "b", matchesNote, run)})
	if err != nil {
		t.Fatal(err)
	}
	a := access(map[string]any{
		"first": struct{}{},
	})
	a.Resolve = func(_ context.Context, id string, _ bool) (any, bool, error) {
		if id == "second" {
			return nil, false, cause
		}
		return struct{}{}, true, nil
	}
	err = d.Deliver(context.Background(), note{}, a)
	if !errors.Is(err, cause) || !errors.Is(err, &event.Error{
		Code: event.ListenerFailed,
	}) || calls != 0 {
		t.Fatalf("prepared partially: calls=%d error=%v", calls, err)
	}
}

type ranked struct {
	order atomic.Int64
	reads atomic.Int64
}

func (r *ranked) Order() int {
	r.reads.Add(1)
	return int(r.order.Load())
}

type priority struct {
	*ranked
}

func (*priority) PriorityOrder() {
}

func TestRuntimeOrderSamplesOwnerOnceAndResamplesNextRound(t *testing.T) {
	normal := &ranked{}
	normal.order.Store(int64(ordered.HighestPrecedence))
	first := &priority{
		ranked: &ranked{},
	}
	first.order.Store(int64(ordered.LowestPrecedence))
	var calls []string
	run := func(id string) bean.EventInvokeFunc {
		return func(context.Context, any, event.Event) error {
			calls = append(calls, id)
			return nil
		}
	}
	override := -100
	method := entry("normal", "b.override", matchesNote, run("override"))
	method.Listener.Order = &override
	d, err := eventdispatch.New([]eventdispatch.Entry{
		entry("normal", "a.normal", matchesNote, run("normal")), method,
		entry("priority", "z.priority", matchesNote, run("priority")),
	})
	if err != nil {
		t.Fatal(err)
	}
	a := access(map[string]any{"normal": normal, "priority": first})
	if err := d.Deliver(context.Background(), note{}, a); err != nil || !slices.Equal(calls, []string{"priority", "normal", "override"}) {
		t.Fatal(calls, err)
	}
	normal.order.Store(100)
	calls = nil
	if err := d.Deliver(context.Background(), note{}, a); err != nil || !slices.Equal(calls, []string{"priority", "override", "normal"}) {
		t.Fatal(calls, err)
	}
	if normal.reads.Load() != 2 || first.reads.Load() != 2 {
		t.Fatal("order was read per method/comparison or cached across rounds")
	}
}

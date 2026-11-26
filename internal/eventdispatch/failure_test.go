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
	"testing"

	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/internal/eventdispatch"
)

func TestFailureAndCancellationStopSubsequentCallbacks(t *testing.T) {
	for _, mode := range []string{"error", "panic", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cause := errors.New("listener failure")
			later := 0
			first := entry("first", "a.first", matchesNote, func(context.Context, any, event.Event) error {
				switch mode {
				case "error":
					return cause
				case "panic":
					panic(cause)
				default:
					cancel()
					return nil
				}
			})
			first.Listener.Source = gs.Source{
				File: "listeners.go",
				Line: 7,
			}
			d, err := eventdispatch.New([]eventdispatch.Entry{first, entry("later", "z.later", matchesNote, func(context.Context, any, event.Event) error {
				later++
				return nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			err = d.Deliver(ctx, note{}, access(map[string]any{
				"first": struct{}{},
				"later": struct{}{},
			}))
			if later != 0 {
				t.Fatal("later callback started after failure/cancellation")
			}
			if mode == "cancel" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			} else {
				var provenance *gs.Error
				if !errors.Is(err, cause) || !errors.Is(err, &event.Error{
					Code: event.ListenerFailed,
				}) || !errors.As(err, &provenance) || provenance.Source.Line != 7 {
					t.Fatal("cause or source lost", err)
				}
			}
		})
	}
}

type panicOrder struct {
	cause error
}

func (p panicOrder) Order() int {
	panic(p.cause)
}

func TestClosingUsesExistingInstancesAndContinuesAfterOrderAndCallbackFailures(t *testing.T) {
	orderCause := errors.New("order panic")
	callbackCause := errors.New("closing callback")
	var calls []string
	d, err := eventdispatch.New([]eventdispatch.Entry{
		entry("bad-order", "a", matchesNote, func(context.Context, any, event.Event) error {
			calls = append(calls, "a")
			return callbackCause
		}),
		entry("other", "z", matchesNote, func(context.Context, any, event.Event) error {
			calls = append(calls, "z")
			return nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	a := access(nil)
	a.Resolve = func(_ context.Context, id string, existing bool) (any, bool, error) {
		if !existing {
			t.Fatal("closing requested instance construction")
		}
		if id == "bad-order" {
			return panicOrder{
				cause: orderCause,
			}, true, nil
		}
		return struct{}{}, true, nil
	}
	err = d.DeliverExisting(context.Background(), note{}, a)
	if !errors.Is(err, orderCause) ||
		!errors.Is(err, callbackCause) ||
		!slices.Equal(calls, []string{"a", "z"}) {
		t.Fatalf("best effort: calls=%v error=%v", calls, err)
	}
}

func TestNoMatchDoesNotResolveUnrelatedLazyOwners(t *testing.T) {
	d, err := eventdispatch.New([]eventdispatch.Entry{entry("unrelated", "a", matchesNote, func(context.Context, any, event.Event) error {
		t.Fatal("unexpected callback")
		return nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	a := access(nil)
	a.Resolve = func(context.Context, string, bool) (any, bool, error) {
		t.Fatal("unrelated receiver resolved")
		return nil, false, nil
	}
	if err := d.Deliver(context.Background(), followup{}, a); err != nil {
		t.Fatal(err)
	}
	if err := d.Deliver(context.Background(), nil, a); !errors.Is(err, &event.Error{
		Code: event.InvalidEvent,
	}) {
		t.Fatal("nil interface admitted", err)
	}
}

func TestMissingExistingReceiverDoesNotPreventOtherClosingCallbacks(t *testing.T) {
	calls := 0
	run := func(context.Context, any, event.Event) error {
		calls++
		return nil
	}
	d, err := eventdispatch.New([]eventdispatch.Entry{entry("missing", "a", matchesNote, run), entry("live", "z", matchesNote, run)})
	if err != nil {
		t.Fatal(err)
	}
	err = d.DeliverExisting(context.Background(), note{}, access(map[string]any{
		"live": struct{}{},
	}))
	if !errors.Is(err, &gs.Error{
		Code: gs.MissingBean,
	}) || calls != 1 {
		t.Fatal(calls, err)
	}
}

func TestDescriptorValidationAndOwnedOverride(t *testing.T) {
	type entryMutationFunc = func(*eventdispatch.Entry)

	for _, change := range []entryMutationFunc{
		func(e *eventdispatch.Entry) {
			e.Bean = " "
		},
		func(e *eventdispatch.Entry) {
			e.Listener.ID = ""
		},
		func(e *eventdispatch.Entry) {
			e.Listener.Event = ""
		},
		func(e *eventdispatch.Entry) {
			e.Listener.Source.Line = -1
		},
		func(e *eventdispatch.Entry) {
			e.Listener.Match = nil
		},
		func(e *eventdispatch.Entry) {
			e.Listener.Invoke = nil
		},
	} {
		invalid := entry("owner", "a", matchesNote, func(context.Context, any, event.Event) error {
			return nil
		})
		change(&invalid)
		if _, err := eventdispatch.New([]eventdispatch.Entry{invalid}); !errors.Is(err, &gs.Error{
			Code: gs.InvalidDefinition,
		}) {
			t.Fatal("invalid descriptor accepted", err)
		}
	}
	var calls []string
	order := -10
	first := entry("owner", "a", matchesNote, func(context.Context, any, event.Event) error {
		calls = append(calls, "first")
		return nil
	})
	first.Listener.Order = &order
	second := entry("owner", "b", matchesNote, func(context.Context, any, event.Event) error {
		calls = append(calls, "second")
		return nil
	})
	input := []eventdispatch.Entry{first, second}
	if _, err := eventdispatch.New([]eventdispatch.Entry{first, first}); !errors.Is(err, &gs.Error{
		Code: gs.InvalidDefinition,
	}) {
		t.Fatal("duplicate declaration accepted", err)
	}
	d, err := eventdispatch.New(input)
	if err != nil {
		t.Fatal(err)
	}
	order = 100
	input[0].Bean = "missing"
	value := &ranked{}
	if err := d.Deliver(context.Background(), note{}, access(map[string]any{"owner": value})); err != nil || !slices.Equal(calls, []string{"first", "second"}) {
		t.Fatal("descriptor snapshot changed", calls, err)
	}
}

func TestMatchPanicPreventsResolution(t *testing.T) {
	cause := errors.New("invalid match adapter")
	run := func(context.Context, any, event.Event) error {
		t.Fatal("unexpected callback")
		return nil
	}
	d, err := eventdispatch.New([]eventdispatch.Entry{
		entry("first", "a", matchesNote, run),
		entry("bad", "b", func(event.Event) bool {
			panic(cause)
		}, run),
	})
	if err != nil {
		t.Fatal(err)
	}
	a := access(nil)
	a.Resolve = func(context.Context, string, bool) (any, bool, error) {
		t.Fatal("resolved before matching completed")
		return nil, false, nil
	}
	if err := d.Deliver(context.Background(), note{}, a); !errors.Is(err, cause) {
		t.Fatal("match panic cause lost", err)
	}
}

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
	"testing"
	"time"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

func TestPublicationPinsScopeUntilCallbackExitsAndPropagatesOwnerCancellation(t *testing.T) {
	entered := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	destroyed := make(chan struct{})
	var e *Engine
	r := listening("listener", "OnEvent", func(ctx context.Context, _ businessEvent) error {
		if _, err := e.Lookup(ctx, gs.Selector{
			Name: "request",
		}); err != nil {
			return err
		}
		close(entered)
		<-ctx.Done()
		close(canceled)
		<-release
		return nil
	})
	request := constant("request")
	request.Definition.Scope = gs.Request
	request.Destroy = func(context.Context, any) error {
		close(destroyed)
		return nil
	}
	e = boot(t, r, request)
	ctx, scope, err := e.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	publication := make(chan error, 1)
	go func() {
		publication <- e.PublishEvent(ctx, businessEvent{})
	}()
	<-entered
	closed := make(chan error, 1)
	go func() {
		closed <- scope.Close(context.Background())
	}()
	<-canceled
	select {
	case <-destroyed:
		t.Error("borrowed request destroyed during listener callback")
	default:
	}
	select {
	case err := <-closed:
		t.Error("scope Close returned before callback exit", err)
	default:
	}
	close(release)
	if err := <-publication; !errors.Is(err, context.Canceled) {
		t.Error("owner cancellation did not reach publication", err)
	}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	<-destroyed
}

func TestPublicationScopeAdmissionUsesNearestKindAndRejectsForeignOwners(t *testing.T) {
	calls := 0
	r := listening("listener", "OnEvent", func(context.Context, businessEvent) error {
		calls++
		return nil
	})
	e := boot(t, r)
	outerCtx, outer, err := e.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	innerCtx, inner, err := e.OpenScope(outerCtx, gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	if err := outer.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := e.PublishEvent(innerCtx, businessEvent{}); err != nil || calls != 1 {
		t.Fatal("closed shadowed owner affected inner scope", calls, err)
	}
	if err := inner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := e.PublishEvent(innerCtx, businessEvent{}); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) || calls != 1 {
		t.Fatal("closed effective owner fell back", calls, err)
	}
	foreign := boot(t)
	foreignCtx, scope, err := foreign.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.PublishEvent(foreignCtx, businessEvent{}); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) || calls != 1 {
		t.Fatal("foreign owner was borrowed", calls, err)
	}
	if err := scope.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPublicationOwnsListenerUntilEngineCloseDrainsIt(t *testing.T) {
	entered, canceled, release, destroyed := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	r := listening("listener", "OnEvent", func(ctx context.Context, _ businessEvent) error {
		close(entered)
		<-ctx.Done()
		close(canceled)
		<-release
		return nil
	})
	r.Destroy = func(context.Context, any) error {
		close(destroyed)
		return nil
	}
	e := boot(t, r)
	publication := make(chan error, 1)
	go func() {
		publication <- e.PublishEvent(context.Background(), businessEvent{})
	}()
	<-entered
	closed := make(chan error, 1)
	go func() {
		closed <- e.Close(context.Background())
	}()
	<-canceled
	select {
	case <-destroyed:
		t.Error("listener destroyed before callback returned")
	default:
	}
	close(release)
	if err := <-publication; !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	<-destroyed
}

func TestListenerScopeSelfCloseFailsWithoutWaitingForItself(t *testing.T) {
	var scope gs.Scope
	r := listening("listener", "OnEvent", func(ctx context.Context, _ businessEvent) error {
		return scope.Close(ctx)
	})
	e := boot(t, r)
	ctx, owned, err := e.OpenScope(context.Background(), gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	scope = owned
	if err := e.PublishEvent(ctx, businessEvent{}); !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatal("self-close did not report its wait cycle", err)
	}
	if err := scope.Close(context.Background()); err != nil {
		t.Fatal("scope cleanup was abandoned after failed self-join", err)
	}
}

func TestForeignScopeRejectionLeavesNoPartialLocalPins(t *testing.T) {
	foreign, err := New(nil, Options{
		Scopes: []gs.ScopeID{"tenant"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := foreign.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := foreign.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	foreignCtx, tenant, err := foreign.OpenScope(context.Background(), "tenant")
	if err != nil {
		t.Fatal(err)
	}
	e := boot(t)
	ctx, local, err := e.OpenScope(foreignCtx, gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.PublishEvent(ctx, businessEvent{}); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal("effective foreign tenant accepted", err)
	}
	// A deadline guards a pin leak; it does not assert scheduling or elapsed time.
	bounded, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := local.Close(bounded); err != nil {
		t.Fatal("failed admission left a partial local pin", err)
	}
	if err := tenant.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

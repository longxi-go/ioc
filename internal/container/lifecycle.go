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

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	applicationClosedDuringStartupMessage      = "application closed during startup"
	closeCannotJoinItsOwnCallbackMessage       = "Close cannot join its own callback"
	startIsSingleAttemptAndNonreentrantMessage = "Start is single-attempt and nonreentrant"
)

// Start is a single attempt: bind the entire graph, initialize eager objects,
// then activate them. Failure compensates every owned allocation before return.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.token(ctx) != nil || e.state != gs.Created {
		e.mu.Unlock()
		return failure(gs.InvalidState, "", startIsSingleAttemptAndNonreentrantMessage, nil)
	}
	if err := ctx.Err(); err != nil {
		e.state = gs.Failed
		e.mu.Unlock()
		return err
	}
	e.state = gs.Starting
	if e.active == 0 {
		e.idle = make(chan struct{})
	}
	e.active++
	t := e.addTask(nil, false, nil)
	e.mu.Unlock()
	call, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(e.base, cancel)
	call = context.WithValue(call, taskKey{}, taskToken{
		e,
		t,
	})
	defer func() {
		stop()
		cancel()
		e.mu.Lock()
		e.removeTask(t)
		e.active--
		if e.active == 0 {
			close(e.idle)
		}
		e.mu.Unlock()
	}()
	err := e.initializeEager(call)

	if err == nil {
		err = e.refreshed(call, t)
	}
	e.mu.Lock()
	if err == nil && e.state == gs.Starting && call.Err() == nil {
		e.state = gs.Running
		e.everRunning = true
		e.mu.Unlock()
		return nil
	}
	if err == nil {
		err = call.Err()
		if err == nil {
			err = failure(gs.InvalidState, "", applicationClosedDuringStartupMessage, nil)
		}
	}
	if e.state != gs.Closing {
		e.state = gs.Failed
	}
	e.mu.Unlock()
	cancel()
	return errors.Join(err, e.rollbackStartup(call, t))
}

// Close rejects new work immediately and cancels in-flight callbacks. A canceled
// caller stops waiting, while the tracked cleanup continues and remains joinable.
func (e *Engine) Close(ctx context.Context) error {
	e.mu.Lock()
	if e.token(ctx) != nil {
		e.mu.Unlock()
		return failure(gs.InvalidState, "", closeCannotJoinItsOwnCallbackMessage, nil)
	}
	if e.closeDone == nil {
		e.closeDone = make(chan struct{})
		e.state = gs.Closing
		e.cancel()
		idle := e.idle
		go e.close(idle)
	}
	done := e.closeDone
	e.mu.Unlock()
	select {
	case <-done:
		e.mu.Lock()
		err := e.closeErr
		e.mu.Unlock()
		return err
	default:
	}
	select {
	case <-done:
		e.mu.Lock()
		err := e.closeErr
		e.mu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Engine) close(idle <-chan struct{}) {
	<-idle
	e.mu.Lock()
	records := e.application.records
	e.application.records = nil
	e.mu.Unlock()
	var errs []error
	if err := e.closingEvent(); err != nil {
		errs = append(errs, err)
	}
	// Quiesce application activations before destroying resources belonging to
	// outstanding requests. Independently closing scopes retain their own runner.
	if err := e.cleanupPhase(context.Background(), cleanupRequest{
		records: records,
		stop:    true,
	}); err != nil {
		errs = append(errs, err)
	}
	e.mu.Lock()
	owners := make([]*owner, 0, len(e.owners))
	for o := range e.owners {
		owners = append(owners, o)
	}
	e.mu.Unlock()
	for _, o := range owners {
		if err := o.Close(context.Background()); err != nil {
			errs = append(errs, err)
		}
	}
	if err := e.cleanupPhase(context.Background(), cleanupRequest{
		records: records,
		destroy: true,
	}); err != nil {
		errs = append(errs, err)
	}
	e.mu.Lock()
	for _, b := range e.bound {
		b.singleton.value = nil
	}
	e.closeErr = errors.Join(errs...)
	e.events = nil
	e.state = gs.Closed
	close(e.closeDone)
	e.mu.Unlock()
}

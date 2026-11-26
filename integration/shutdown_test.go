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

package integration_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func TestCloseDoesNotPublishLateFactoryResult(t *testing.T) {
	r := registration("late")
	r.Definition.Lazy = true
	entered, release := make(chan struct{}), make(chan struct{})
	var destroyed atomic.Int32
	r.Create = func(context.Context, bean.Ports) (any, error) {
		close(entered)
		<-release
		return &item{
			name: "late",
		}, nil
	}
	r.Destroy = func(context.Context, any) error {
		destroyed.Add(1)
		return nil
	}
	app := newApplication(t, []bean.Registration{r})
	base := context.Background()
	if err := app.Start(base); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		value, err := app.GetBean[*item](base)
		if value != nil && err == nil {
			result <- errors.New("late object published")
		} else {
			result <- err
		}
	}()
	<-entered
	closeCtx, cancel := context.WithTimeout(base, 20*time.Millisecond)
	defer cancel()
	err := app.Close(closeCtx)
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("close wait ignored deadline: %v", err)
	}
	if err := <-result; err == nil {
		t.Fatal("late result accepted")
	}
	doneCtx, stop := context.WithTimeout(base, time.Second)
	defer stop()
	if err := app.Close(doneCtx); err != nil {
		t.Fatal(err)
	}
	if destroyed.Load() != 1 {
		t.Fatal(destroyed.Load())
	}
}

func TestStartAndCloseCallbacksCannotWaitForThemselves(t *testing.T) {
	for _, stage := range []string{"start", "destroy"} {
		t.Run(stage, func(t *testing.T) {
			r := registration("callback")
			var callbackErr error
			var app gs.ApplicationContext
			if stage == "start" {
				r.Start = func(ctx context.Context, _ any) error {
					callbackErr = app.Start(ctx)
					return nil
				}
			} else {
				r.Destroy = func(ctx context.Context, _ any) error {
					callbackErr = app.Close(ctx)
					return nil
				}
			}
			// The closure target is the context that owns this registration.
			app = newApplication(t, []bean.Registration{r})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := app.Start(ctx); err != nil {
				t.Fatal(err)
			}
			if err := app.Close(ctx); err != nil {
				t.Fatal(err)
			}
			if !errors.Is(callbackErr, &gs.Error{
				Code: gs.InvalidState,
			}) {
				t.Fatalf("self wait: %v", callbackErr)
			}
		})
	}
}

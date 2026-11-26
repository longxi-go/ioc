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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/longxi-go/gosr/pkg/event"
	"github.com/longxi-go/ioc"
)

// OrderPlaced is an explicit business fact without an application reference.
type OrderPlaced struct {
	event.BaseEvent
	ID string
}

// OrderService depends only on publication; it does not locate its receivers.
// +ioc:component
type OrderService struct {
	// +ioc:inject
	events event.Publisher
}

// Place illustrates publication after a business operation has completed.
func (s *OrderService) Place(ctx context.Context, id string) error {
	return s.events.PublishEvent(ctx, OrderPlaced{
		ID: id,
	})
}

// Audit is a managed typed listener whose instance supplies relative order.
// +ioc:component:name=orderAudit
// +ioc:listener
type Audit struct{}

// Order leaves an insertion point before ordinary notification observers.
func (*Audit) Order() int {
	return 100
}

// OnEvent handles only the declared fact through a generated direct adapter.
func (*Audit) OnEvent(_ context.Context, placed OrderPlaced) error {
	fmt.Println("audit", placed.ID)
	return nil
}

// Notifications demonstrates a private, explicitly annotated method.
// +ioc:component
type Notifications struct{}

// +ioc:listen
// +ioc:order=200
func (*Notifications) onPlaced(_ context.Context, placed OrderPlaced) error {
	fmt.Println("notify", placed.ID)
	return nil
}

func run() (err error) {
	ctx := context.Background()
	app, err := ioc.NewApplicationContext(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, app.Close(context.WithoutCancel(ctx)))
	}()
	if err := app.Start(ctx); err != nil {
		return err
	}
	service, err := app.GetBean[*OrderService](ctx)
	if err != nil {
		return err
	}
	return service.Place(ctx, "order-42")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

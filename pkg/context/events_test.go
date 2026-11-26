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

package context_test

import (
	"context"
	"errors"
	"testing"

	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/ioc/pkg/catalog"
	application "github.com/longxi-go/ioc/pkg/context"
)

type (
	placed struct {
		event.BaseEvent
	}

	eventReceiver struct {
		publisher event.Publisher
		count     int
	}
)

func TestInjectedPublisherIsLocalAndDoesNotDependOnListeners(t *testing.T) {
	c := catalog.New()
	r := bean.Registration{
		Definition: gs.BeanDefinition{
			ID:        "receiver",
			Name:      "receiver",
			Scope:     gs.Singleton,
			Contracts: []gs.Contract{"receiver"},
			Dependencies: []gs.Dependency{
				{
					Selector: gs.Selector{
						Contract: bean.EventPublisherContract,
					},
					Access:      gs.Direct,
					Cardinality: gs.One,
				},
			},
		},
		Create: func(ctx context.Context, ports bean.Ports) (any, error) {
			publisher, err := bean.Dependency[event.Publisher](ctx, ports, 0)
			return &eventReceiver{
				publisher: publisher,
			}, err
		},
		Listeners: []bean.Listener{
			{
				ID:    "receiver.OnEvent",
				Event: "placed",
				Match: func(value event.Event) bool {
					_, ok := value.(placed)
					return ok
				},
				Invoke: func(_ context.Context, object any, _ event.Event) error {
					object.(*eventReceiver).count++
					return nil
				},
			},
		},
	}
	c.Contribute(bean.Batch{
		ID:            "events",
		Schema:        bean.SchemaVersion,
		Registrations: []bean.Registration{r},
	})
	ctx := context.Background()
	apps := make([]*application.Application, 2)
	listeners := make([]*eventReceiver, 2)
	for i := range apps {
		a, err := application.NewApplicationContext(context.Background(), application.WithCatalog(c))
		if err != nil {
			t.Fatal(err)
		}
		apps[i] = a
		t.Cleanup(func() {
			if err := a.Close(ctx); err != nil {
				t.Error(err)
			}
		})
		if err := a.Start(ctx); err != nil {
			t.Fatal("publisher/listener invented construction cycle", err)
		}
		value, err := a.Lookup(ctx, gs.Selector{
			Name: "receiver",
		})
		if err != nil {
			t.Fatal(err)
		}
		listeners[i] = value.(*eventReceiver)
		if _, leaked := listeners[i].publisher.(gs.BeanFactory); leaked {
			t.Fatal("narrow publisher exposed engine capabilities")
		}
	}
	if listeners[0].publisher == listeners[1].publisher {
		t.Fatal("shared catalog retained an application publisher instance")
	}
	if err := listeners[0].publisher.PublishEvent(ctx, placed{}); err != nil {
		t.Fatal(err)
	}
	if err := apps[0].PublishEvent(ctx, placed{}); err != nil || listeners[0].count != 2 || listeners[1].count != 0 {
		t.Fatal("publication escaped owner context", listeners[0].count, listeners[1].count, err)
	}
}

func TestDefaultPublisherCannotBeDecorated(t *testing.T) {
	_, err := application.NewApplicationContext(context.Background(), application.WithCatalog(catalog.New()), application.WithDecorator(bean.Decorator{
		Contract: bean.EventPublisherContract,
		Wrap: func(context.Context, any) (any, error) {
			t.Fatal("reserved publisher decorated")
			return nil, nil
		},
	}))
	if !errors.Is(err, &gs.Error{
		Code: gs.InvalidDefinition,
	}) {
		t.Fatal("reserved decoration accepted", err)
	}
}

type alternativePublisher struct {
	calls int
}

func (p *alternativePublisher) PublishEvent(context.Context, event.Event) error {
	p.calls++
	return nil
}

func TestNamedPublisherAlternativeDoesNotReplaceLocalDefault(t *testing.T) {
	other := &alternativePublisher{}
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:     "alternative",
		Schema: bean.SchemaVersion,
		Registrations: []bean.Registration{
			{
				Definition: gs.BeanDefinition{
					ID:        "remote",
					Name:      "remote",
					Scope:     gs.Singleton,
					Contracts: []gs.Contract{bean.EventPublisherContract},
				},
				Create: func(context.Context, bean.Ports) (any, error) {
					return other, nil
				},
			},
		},
	})
	a, err := application.NewApplicationContext(context.Background(), application.WithCatalog(c))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := a.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	value, err := a.Lookup(ctx, gs.Selector{
		Contract: bean.EventPublisherContract,
		Name:     "remote",
	})
	if err != nil || value != other {
		t.Fatal("named alternative inaccessible", value, err)
	}
	local, err := a.Lookup(ctx, gs.Selector{
		Contract: bean.EventPublisherContract,
	})
	if err != nil || local == other {
		t.Fatal("alternative replaced primary default", local, err)
	}
	if err := a.PublishEvent(ctx, placed{}); err != nil || other.calls != 0 {
		t.Fatal("local publish routed into named alternative", other.calls, err)
	}
}

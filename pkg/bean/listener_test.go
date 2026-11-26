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

package bean_test

import (
	"context"
	"errors"
	"testing"

	"github.com/longxi-go/gosr/pkg/event"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

func listenerRegistration() bean.Registration {
	order := 100
	return bean.Registration{
		Definition: gs.BeanDefinition{
			ID:        "orders/audit",
			Name:      "audit",
			Contracts: []gs.Contract{"Audit"},
			Scope:     gs.Singleton,
		},
		Create: func(context.Context, bean.Ports) (any, error) {
			panic("validation must not construct")
		},
		Listeners: []bean.Listener{
			{
				ID:    "orders/audit.OnEvent",
				Event: "OrderPlaced",
				Order: &order,
				Source: gs.Source{
					File: "audit.go",
					Line: 7,
				},
				Match: func(event.Event) bool {
					panic("validation must not match")
				},
				Invoke: func(context.Context, any, event.Event) error {
					panic("validation must not invoke")
				},
			},
		},
	}
}

func TestRegistrationValidationDoesNotExecuteAdapters(t *testing.T) {
	if err := listenerRegistration().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrationRejectsInvalidListenerDeclarations(t *testing.T) {
	cases := map[string]func(*bean.Registration){
		"identity": func(r *bean.Registration) {
			r.Listeners[0].ID = " "
		},
		"event": func(r *bean.Registration) {
			r.Listeners[0].Event = ""
		},
		"match": func(r *bean.Registration) {
			r.Listeners[0].Match = nil
		},
		"invoke": func(r *bean.Registration) {
			r.Listeners[0].Invoke = nil
		},
		"source": func(r *bean.Registration) {
			r.Listeners[0].Source.Line = -1
		},
		"duplicate": func(r *bean.Registration) {
			r.Listeners = append(r.Listeners, r.Listeners[0])
		},
		"request owner": func(r *bean.Registration) {
			r.Definition.Scope = gs.Request
		},
		"prototype owner": func(r *bean.Registration) {
			r.Definition.Scope = gs.Prototype
		},
		"factory": func(r *bean.Registration) {
			r.Create = nil
		},
		"definition": func(r *bean.Registration) {
			r.Definition.Contracts = nil
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r := listenerRegistration()
			change(&r)
			if err := r.Validate(); !errors.Is(err, &gs.Error{
				Code: gs.InvalidDefinition,
			}) {
				t.Fatalf("invalid declaration accepted: %v", err)
			}
		})
	}
}

func TestReservedPublisherIdentityCannotBeContributed(t *testing.T) {
	for _, field := range []string{"id", "name", "alias"} {
		t.Run(field, func(t *testing.T) {
			r := listenerRegistration()
			switch field {
			case "id":
				r.Definition.ID = bean.EventPublisherID
			case "name":
				r.Definition.Name = bean.EventPublisherName
			case "alias":
				r.Definition.Aliases = []string{bean.EventPublisherName}
			}
			if err := r.Validate(); !errors.Is(err, &gs.Error{
				Code: gs.InvalidDefinition,
			}) {
				t.Fatalf("reserved publisher identity accepted: %v", err)
			}
		})
	}
}

func TestRegistrationCloneOwnsListenerMetadata(t *testing.T) {
	r := listenerRegistration()
	copy := r.Clone()
	copy.Listeners[0].ID = "changed"
	copy.Listeners[0].Source.File = "changed.go"
	*copy.Listeners[0].Order = -100
	if r.Listeners[0].ID != "orders/audit.OnEvent" ||
		r.Listeners[0].Source.File != "audit.go" ||
		*r.Listeners[0].Order != 100 {
		t.Fatal("clone shares listener metadata or optional order pointer")
	}
	copy.Listeners = append(copy.Listeners, bean.Listener{})
	if len(r.Listeners) != 1 {
		t.Fatal("clone shares listener slice")
	}
	if (bean.Registration{}).Clone().Listeners != nil {
		t.Fatal("clone changed absent listener shape")
	}
}

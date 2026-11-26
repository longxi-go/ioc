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

package compiler

import gs "github.com/longxi-go/gosr/pkg/ioc"

// Render views contain Go spellings already chosen by type analysis. Templates
// own presentation; they never resolve packages or select dependency candidates.
type (
	importView struct {
		Path, Alias string
	}

	fileView struct {
		Header, Package, Constraint, Body string
		Imports                           []importView
	}
)

type packageView struct {
	ID            string
	Schema        int
	Registrations []registrationView
}

type registrationView struct {
	Definition                                 gs.BeanDefinition
	Conditions                                 []condition
	Result, Component, Function, CallArguments string
	ReturnsError                               bool
	PointerResult                              bool
	Arguments                                  []argumentView
	Fields                                     []assignmentView
	Hooks                                      []lifecycleHook
	Listeners                                  []listenerView
}

type (
	argumentView struct {
		Name, Expression string
	}

	assignmentView struct {
		Field, Value string
	}

	listenerView struct {
		ID, Event, EventType, EventInterface, Receiver, Method string
		Source                                                 gs.Source
		HasOrder                                               bool
		Order                                                  int
	}
)

type queryView struct {
	ID            string
	DefinitionIDs []string
	Cases         []queryCase
}

type (
	queryCase struct {
		Type, Contract string
	}

	resolutionView struct {
		Provider                                      bool
		Type, ElementType, Context, Inner, Conversion string
		Port                                          int
	}
)

type conversionView struct {
	Optional, Many      bool
	Type, Values, Inner string
}

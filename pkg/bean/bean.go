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

package bean

import (
	"context"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

// SchemaVersion identifies the generated recipe ABI.
const SchemaVersion = 4

// Ports resolves a validated dependency position using the invocation context.
type Ports interface {
	Resolve(context.Context, int) ([]any, error)
}

// Factory assembles an object. A nonnil result transfers cleanup ownership even
// when accompanied by an error; a nil result leaves cleanup to the constructor.
// Generated adapters normalize concrete nil pointers before boxing. Handwritten
// adapters and constructors returning interfaces must enforce their own nonnil
// object invariant; no runtime reflection inspects opaque interface values.
type Factory func(context.Context, Ports) (any, error)

// Hook is generated lifecycle dispatch without runtime method discovery.
type Hook func(context.Context, any) error

// Registration combines portable metadata with static executable adapters.
type Registration struct {
	Definition                 gs.BeanDefinition
	Create                     Factory
	Init, Start, Stop, Destroy Hook
	Accept                     AcceptFunc
	Conditions                 []Condition
	Listeners                  []Listener
}

// Batch is an atomic generated contribution; an invalid member rejects the batch.
type Batch struct {
	ID            string
	Schema        int
	Registrations []Registration
}

// Query dispatches a generated output type to its canonical contract. Collection
// is explicit so a []T bean remains distinguishable from a collection of T.
type Query func(context.Context, gs.BeanFactory, any, []string, bool) error

// Plan fixes application discovery closure independently of global contributions.
type Plan struct {
	ID            string
	DefinitionIDs []string
	Query         Query
}

// Decorator replaces a value before initialization and contract validation.
// It must return the same owned object or assume responsibility for the wrapped one.
type Decorator struct {
	Contract gs.Contract
	Wrap     DecoratorFunc
}

// Dependency resolves one statically bound port and verifies the generated cast.
func Dependency[T any](ctx context.Context, p Ports, index int) (T, error) {
	var zero T
	vs, err := p.Resolve(ctx, index)
	if err != nil {
		return zero, err
	}
	if len(vs) == 0 {
		return zero, &gs.Error{
			Code: gs.MissingBean,
		}
	}
	if len(vs) != 1 {
		return zero, &gs.Error{
			Code: gs.AmbiguousBean,
		}
	}
	v, ok := vs[0].(T)
	if !ok {
		return zero, &gs.Error{
			Code: gs.ContractMismatch,
		}
	}
	return v, nil
}

// Dependencies retains the runtime's stable collection ordering.
func Dependencies[T any](ctx context.Context, p Ports, index int) ([]T, error) {
	vs, err := p.Resolve(ctx, index)
	if err != nil {
		return nil, err
	}
	out := make([]T, len(vs))
	for i, v := range vs {
		typed, ok := v.(T)
		if !ok {
			return nil, &gs.Error{
				Code: gs.ContractMismatch,
			}
		}
		out[i] = typed
	}
	return out, nil
}

// OptionalDependency permits only a genuinely empty bound port.
func OptionalDependency[T any](ctx context.Context, p Ports, index int) (gs.Optional[T], error) {
	vs, err := p.Resolve(ctx, index)
	if err != nil {
		return gs.Optional[T]{}, err
	}
	if len(vs) == 0 {
		return gs.Optional[T]{}, nil
	}
	v, err := Dependency[T](ctx, fixedPorts(vs), 0)
	return gs.Optional[T]{
		Value:   v,
		Present: err == nil,
	}, err
}

type fixedPorts []any

func (p fixedPorts) Resolve(context.Context, int) ([]any, error) {
	return p, nil
}

// Assign is the static generated query adapter for a single contract.
func Assign[T any](ctx context.Context, f gs.BeanFactory, out *T, s gs.Selector) error {
	v, err := f.Lookup(ctx, s)
	if err != nil {
		return err
	}
	typed, ok := v.(T)
	if !ok {
		return &gs.Error{
			Code: gs.ContractMismatch,
		}
	}
	*out = typed
	return nil
}

// AssignAll adapts ordered values without reflection or runtime type discovery.
func AssignAll[T any](ctx context.Context, f gs.BeanFactory, out *[]T, s gs.Selector) error {
	vs, err := f.LookupAll(ctx, s)
	if err != nil {
		return err
	}
	values := make([]T, len(vs))
	for i, v := range vs {
		typed, ok := v.(T)
		if !ok {
			return &gs.Error{
				Code: gs.ContractMismatch,
			}
		}
		values[i] = typed
	}
	*out = values
	return nil
}

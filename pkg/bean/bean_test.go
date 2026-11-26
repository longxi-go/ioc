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

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

type ports struct {
	values []any
	err    error
}

func (p ports) Resolve(context.Context, int) ([]any, error) {
	return p.values, p.err
}

func TestTypedPorts(t *testing.T) {
	ctx := context.Background()
	v, err := bean.Dependency[int](ctx, ports{
		values: []any{7},
	}, 0)
	if err != nil || v != 7 {
		t.Fatalf("value=%v error=%v", v, err)
	}
	_, err = bean.Dependency[int](ctx, ports{
		values: []any{"bad"},
	}, 0)
	if !errors.Is(err, &gs.Error{
		Code: gs.ContractMismatch,
	}) {
		t.Fatal(err)
	}
	optional, err := bean.OptionalDependency[int](ctx, ports{}, 0)
	if err != nil || optional.Present {
		t.Fatal(optional, err)
	}
	_, err = bean.Dependency[int](ctx, ports{}, 0)
	if !errors.Is(err, &gs.Error{
		Code: gs.MissingBean,
	}) {
		t.Fatal(err)
	}
	values, err := bean.Dependencies[int](ctx, ports{
		values: []any{1, 2},
	}, 0)
	if err != nil || len(values) != 2 {
		t.Fatal(values, err)
	}
}

type factory struct {
	values []any
	err    error
}

func (f factory) Lookup(context.Context, gs.Selector) (any, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.values[0], nil
}

func (f factory) LookupAll(context.Context, gs.Selector) ([]any, error) {
	return f.values, f.err
}

func TestAssignmentsPreserveOutputOnError(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("lookup")
	for _, f := range []factory{
		{
			err: sentinel,
		},
		{
			values: []any{"wrong"},
		},
	} {
		value := 9
		err := bean.Assign(ctx, f, &value, gs.Selector{
			Name: "x",
		})
		if err == nil || value != 9 {
			t.Fatal(value, err)
		}
		values := []int{9}
		err = bean.AssignAll(ctx, f, &values, gs.Selector{
			Name: "x",
		})
		if err == nil || len(values) != 1 || values[0] != 9 {
			t.Fatal(values, err)
		}
	}
	var value int
	if err := bean.Assign(ctx, factory{
		values: []any{7},
	}, &value, gs.Selector{
		Name: "x",
	}); err != nil || value != 7 {
		t.Fatal(value, err)
	}
	var values []int
	if err := bean.AssignAll(ctx, factory{
		values: []any{2, 3},
	}, &values, gs.Selector{
		Name: "x",
	}); err != nil || len(values) != 2 || values[1] != 3 {
		t.Fatal(values, err)
	}
}

func TestPortErrorsAndTypedZeroRemainDistinct(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("construction")
	if _, err := bean.Dependency[int](ctx, ports{
		err: sentinel,
	}, 0); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if _, err := bean.Dependencies[int](ctx, ports{
		err: sentinel,
	}, 0); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if _, err := bean.OptionalDependency[int](ctx, ports{
		err: sentinel,
	}, 0); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if _, err := bean.Dependency[int](ctx, ports{
		values: []any{1, 2},
	}, 0); !errors.Is(err, &gs.Error{
		Code: gs.AmbiguousBean,
	}) {
		t.Fatal(err)
	}
	if _, err := bean.Dependencies[int](ctx, ports{
		values: []any{"bad"},
	}, 0); !errors.Is(err, &gs.Error{
		Code: gs.ContractMismatch,
	}) {
		t.Fatal(err)
	}
	optional, err := bean.OptionalDependency[*int](ctx, ports{
		values: []any{(*int)(nil)},
	}, 0)
	if err != nil || !optional.Present || optional.Value != nil {
		t.Fatal(optional, err)
	}
	if _, err = bean.OptionalDependency[int](ctx, ports{
		values: []any{"bad"},
	}, 0); !errors.Is(err, &gs.Error{
		Code: gs.ContractMismatch,
	}) {
		t.Fatal(err)
	}
}

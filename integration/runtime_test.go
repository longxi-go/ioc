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
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/ioc/pkg/catalog"
	appcontext "github.com/longxi-go/ioc/pkg/context"
)

type item struct {
	name string
}

func registration(id string) bean.Registration {
	return bean.Registration{
		Definition: gs.BeanDefinition{
			ID:        id,
			Name:      id,
			Contracts: []gs.Contract{"item"},
			Scope:     gs.Singleton,
		},
		Create: func(context.Context, bean.Ports) (any, error) {
			return &item{
				name: id,
			}, nil
		},
		Accept: func(v any) bool {
			x, ok := v.(*item)
			return ok && x != nil
		},
	}
}

func query(ctx context.Context, f gs.BeanFactory, out any, names []string, many bool) error {
	selector := gs.Selector{
		Contract: "item",
	}
	if len(names) > 0 {
		selector.Name = names[0]
	}
	switch target := out.(type) {
	case **item:
		if !many {
			return bean.Assign(ctx, f, target, selector)
		}
	case *[]*item:
		if many {
			return bean.AssignAll(ctx, f, target, selector)
		}
	}
	return &gs.Error{
		Code: gs.UnsupportedContract,
	}
}

func newApplication(t *testing.T, rs []bean.Registration, options ...appcontext.Option) *appcontext.Application {
	t.Helper()
	c := catalog.New()
	ids := make([]string, 0, len(rs))
	for _, r := range rs {
		ids = append(ids, r.Definition.ID)
	}
	c.Contribute(bean.Batch{
		ID:            "integration",
		Schema:        bean.SchemaVersion,
		Registrations: rs,
	})
	c.Install(bean.Plan{
		ID:            "integration",
		DefinitionIDs: ids,
		Query:         query,
	})
	app, err := appcontext.NewApplicationContext(context.Background(), append([]appcontext.Option{appcontext.WithCatalog(c)}, options...)...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := app.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	return app
}

func TestIndependentContextsAndTypedSelection(t *testing.T) {
	read, write := registration("read"), registration("write")
	read.Definition.Primary = true
	read.Definition.Aliases = []string{"main"}
	read.Definition.Order = -1
	a := newApplication(t, []bean.Registration{write, read})
	b := newApplication(t, []bean.Registration{write, read})
	ctx := context.Background()
	if err := a.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Start(ctx); err != nil {
		t.Fatal(err)
	}
	first, err := a.GetBean[*item](ctx)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := a.GetBean[*item](ctx, "main")
	if err != nil || alias != first {
		t.Fatalf("alias identity %v", err)
	}
	other, err := b.GetBean[*item](ctx)
	if err != nil || other == first {
		t.Fatalf("contexts share singleton %v", err)
	}
	values, err := a.GetBeans[*item](ctx)
	if err != nil ||
		len(values) != 2 ||
		values[0].name != "read" ||
		values[1].name != "write" {
		t.Fatalf("collection=%v error=%v", values, err)
	}
	if _, err := a.Lookup(ctx, gs.Selector{
		Name:     "read",
		Contract: "other",
	}); !errors.Is(err, &gs.Error{
		Code: gs.ContractMismatch,
	}) {
		t.Fatalf("wrong named contract %v", err)
	}
	if _, err := a.Lookup(ctx, gs.Selector{}); !errors.Is(err, &gs.Error{
		Code: gs.InvalidSelector,
	}) {
		t.Fatal(err)
	}
	if _, err := a.GetBean[int](ctx); !errors.Is(err, &gs.Error{
		Code: gs.UnsupportedContract,
	}) {
		t.Fatal(err)
	}
	view := a.Definitions()
	view[0].Contracts[0] = "mutated"
	if a.Definitions()[0].Contracts[0] == "mutated" {
		t.Fatal("mutable application metadata")
	}
}

func TestDirectCycleFailsBeforeConstruction(t *testing.T) {
	a, b := registration("a"), registration("b")
	calls := 0
	for _, r := range []*bean.Registration{&a, &b} {
		target := "a"
		if r.Definition.ID == "a" {
			target = "b"
		}
		r.Definition.Dependencies = []gs.Dependency{
			{
				Selector: gs.Selector{
					Name: target,
				},
				Access:      gs.Direct,
				Cardinality: gs.One,
				Source: gs.Source{
					File: "cycle.go",
					Line: 7,
				},
			},
		}
		r.Create = func(context.Context, bean.Ports) (any, error) {
			calls++
			return &item{}, nil
		}
	}
	app := newApplication(t, []bean.Registration{a, b})
	if err := app.Start(context.Background()); !errors.Is(err, &gs.Error{
		Code: gs.DependencyCycle,
	}) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("allocated before graph check")
	}
}

func TestFailedActivationCompensatesAttemptAndAllocation(t *testing.T) {
	var events []string
	r := registration("service")
	r.Init = func(context.Context, any) error {
		events = append(events, "init")
		return nil
	}
	r.Start = func(context.Context, any) error {
		events = append(events, "start")
		return errors.New("activation failed")
	}
	r.Stop = func(context.Context, any) error {
		events = append(events, "stop")
		return nil
	}
	r.Destroy = func(context.Context, any) error {
		events = append(events, "destroy")
		return nil
	}
	app := newApplication(t, []bean.Registration{r})
	if err := app.Start(context.Background()); err == nil {
		t.Fatal("failed start accepted")
	}
	if app.State() != gs.Failed {
		t.Fatal(app.State())
	}
	if err := app.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(events, []string{"init", "start", "stop", "destroy"}) {
		t.Fatal(events)
	}
}

func TestExplicitScopeReuseAndIsolation(t *testing.T) {
	r := registration("request")
	r.Definition.Scope = gs.Request
	var destroyed atomic.Int32
	r.Destroy = func(context.Context, any) error {
		destroyed.Add(1)
		return nil
	}
	app := newApplication(t, []bean.Registration{r})
	base := context.Background()
	if err := app.Start(base); err != nil {
		t.Fatal(err)
	}
	if _, err := app.GetBean[*item](base); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
	first, s1, err := app.OpenScope(base, gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	second, s2, err := app.OpenScope(base, gs.Request)
	if err != nil {
		t.Fatal(err)
	}
	x, err := app.GetBean[*item](first)
	if err != nil {
		t.Fatal(err)
	}
	y, err := app.GetBean[*item](first)
	if err != nil || x != y {
		t.Fatal("scope cache failed", err)
	}
	z, err := app.GetBean[*item](second)
	if err != nil || x == z {
		t.Fatal("scope isolation failed", err)
	}
	if err := s1.Close(base); err != nil {
		t.Fatal(err)
	}
	if _, err := app.GetBean[*item](first); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
	if err := s2.Close(base); err != nil {
		t.Fatal(err)
	}
	if destroyed.Load() != 2 {
		t.Fatal(destroyed.Load())
	}
}

func TestResourcePrototypeNeedsLease(t *testing.T) {
	r := registration("prototype")
	r.Definition.Scope = gs.Prototype
	var closed atomic.Int32
	r.Destroy = func(context.Context, any) error {
		closed.Add(1)
		return nil
	}
	app := newApplication(t, []bean.Registration{r})
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := app.GetBean[*item](ctx); !errors.Is(err, &gs.Error{
		Code: gs.ScopeViolation,
	}) {
		t.Fatal(err)
	}
	lease, err := app.AcquireBean[*item](ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Value() == nil {
		t.Fatal("nil acquisition")
	}
	if err := lease.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if closed.Load() != 1 {
		t.Fatal(closed.Load())
	}
}

func TestConcurrentLazySingletonHasOneOwner(t *testing.T) {
	r := registration("lazy")
	r.Definition.Lazy = true
	var creates atomic.Int32
	r.Create = func(context.Context, bean.Ports) (any, error) {
		creates.Add(1)
		return &item{}, nil
	}
	app := newApplication(t, []bean.Registration{r})
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	values := make(chan *item, 16)
	failures := make(chan error, 16)
	var wait sync.WaitGroup
	for range 16 {
		wait.Go(func() {
			value, err := app.GetBean[*item](ctx)
			if err != nil {
				failures <- err
			} else {
				values <- value
			}
		})
	}
	wait.Wait()
	close(values)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	var first *item
	for value := range values {
		if first == nil {
			first = value
		} else if first != value {
			t.Fatal("multiple cached instances")
		}
	}
	if creates.Load() != 1 {
		t.Fatal(creates.Load())
	}
}

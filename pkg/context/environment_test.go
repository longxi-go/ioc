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

	env "github.com/longxi-go/gosr/pkg/environment"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
	"github.com/longxi-go/ioc/pkg/catalog"
	app "github.com/longxi-go/ioc/pkg/context"
	"github.com/longxi-go/nemo"
	ne "github.com/longxi-go/nemo/pkg/environment"
	"github.com/longxi-go/nemo/pkg/maptypes"
	"github.com/longxi-go/nemo/pkg/source"
)

func TestEnvironmentBootstrap(t *testing.T) {
	t.Setenv("IOC_ENVIRONMENT_TEST", "process")
	properties := map[string]string{"ioc.environment.test": "programmatic"}
	option := app.WithProperties(properties)
	properties["ioc.environment.test"] = "changed"
	application, err := app.NewApplicationContext(context.Background(), app.WithCatalog(catalog.New()), option)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := application.Close(context.Background()); err != nil {
			t.Error(err)
		}
	}()
	p, found, err := application.Environment().Lookup("ioc.environment.test")
	if err != nil || !found {
		t.Fatal(found, err)
	}
	if text, _ := p.Value.Scalar(); text != "programmatic" {
		t.Fatal(text)
	}
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	injected, err := application.Lookup(context.Background(), gs.Selector{
		Contract: bean.EnvironmentContract,
	})
	if err != nil || injected != application.Environment() {
		t.Fatal(injected, err)
	}
}

func TestCustomEnvironmentAndFrozenConditions(t *testing.T) {
	provider := &changingProvider{
		value: "yes",
	}
	owner, err := nemo.New(context.Background(), ne.WithProviders(provider))
	if err != nil {
		t.Fatal(err)
	}
	c := catalog.New()
	c.Contribute(bean.Batch{
		ID:     "test",
		Schema: bean.SchemaVersion,
		Registrations: []bean.Registration{
			{
				Definition: gs.BeanDefinition{
					ID:        "candidate",
					Name:      "candidate",
					Scope:     gs.Singleton,
					Contracts: []gs.Contract{"string"},
				},
				Conditions: []bean.Condition{
					{
						Property: &bean.PropertyCondition{
							Names:       []string{"enabled"},
							HavingValue: new("yes"),
						},
					},
				},
				Create: func(context.Context, bean.Ports) (any, error) {
					return "selected", nil
				},
			},
		},
	})
	c.Install(bean.Plan{
		ID:            "test",
		DefinitionIDs: []string{"candidate"},
	})
	application, err := app.NewApplicationContext(context.Background(), app.WithCatalog(c), app.WithEnvironment(owner))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := application.Close(context.Background()); err != nil {
			t.Error(err)
		}
	}()
	if application.Environment() != owner {
		t.Fatal("owner identity lost")
	}
	provider.value = "no"
	if err := owner.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := application.Lookup(context.Background(), gs.Selector{
		Contract: "string",
	})
	if err != nil || result != "selected" {
		t.Fatal(result, err)
	}
	report, err := application.ConditionReport()
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range report.Candidates {
		if candidate.Bean == "candidate" && candidate.Outcome != bean.Accepted {
			t.Fatal(candidate)
		}
	}
}

type changingProvider struct {
	value string
}

func (p *changingProvider) Fetch(context.Context) (env.PropertySource, error) {
	return source.NewMap("changing", maptypes.ValueMap{
		"enabled": env.String(p.value),
	})
}

func TestEnvironmentBootstrapFailures(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := app.NewApplicationContext(cancelled, app.WithCatalog(catalog.New()))
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	//nolint:staticcheck // SA1012: deliberately exercise nil-context rejection.
	if _, err := app.NewApplicationContext(nil); err == nil {
		t.Fatal("nil context accepted")
	}
	if _, err := app.NewApplicationContext(context.Background(), app.WithEnvironment(nil)); err == nil {
		t.Fatal("nil environment accepted")
	}
	owner, err := nemo.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range []app.Option{app.WithProperties(nil), app.WithEnvironmentOptions()} {
		if _, err := app.NewApplicationContext(context.Background(), app.WithEnvironment(owner), option); err == nil {
			t.Fatal("conflicting options accepted")
		}
	}
}

type panickingEnvironment struct {
	app.SnapshotEnvironment
	cause error
}

func (e panickingEnvironment) Snapshot() env.Snapshot {
	panic(e.cause)
}

func TestBootstrapPanicPreservesCause(t *testing.T) {
	cause := context.Canceled
	_, err := app.NewApplicationContext(context.Background(), app.WithCatalog(catalog.New()), app.WithEnvironment(panickingEnvironment{
		cause: cause,
	}))
	if !errors.Is(err, cause) {
		t.Fatalf("lost panic cause: %v", err)
	}
}

func TestEnvironmentProviderPrecedenceAndLoadCancellation(t *testing.T) {
	t.Setenv("IOC_ENVIRONMENT_TEST", "process")
	explicit, err := source.NewMap("explicit", maptypes.ValueMap{
		"ioc.environment.test": env.String("explicit"),
	})
	if err != nil {
		t.Fatal(err)
	}
	application, err := app.NewApplicationContext(context.Background(), app.WithCatalog(catalog.New()), app.WithEnvironmentOptions(ne.WithProviders(source.Static(explicit))))
	if err != nil {
		t.Fatal(err)
	}
	property, present, err := application.Environment().Lookup("ioc.environment.test")
	text, _ := property.Value.Scalar()
	if err != nil || !present || text != "explicit" {
		t.Fatal(property, present, err)
	}
	if err := application.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	_, err = app.NewApplicationContext(ctx, app.WithCatalog(catalog.New()), app.WithEnvironmentOptions(ne.WithProviders(cancelProvider{
		cancel: cancel,
	})))
	if !errors.Is(err, context.Canceled) {
		t.Fatal("provider did not receive construction context", err)
	}
}

type cancelProvider struct {
	cancel context.CancelFunc
}

func (p cancelProvider) Fetch(ctx context.Context) (env.PropertySource, error) {
	p.cancel()
	return nil, ctx.Err()
}

func TestReservedEnvironment(t *testing.T) {
	for _, definition := range []gs.BeanDefinition{
		{
			ID:    bean.EnvironmentID,
			Name:  "user",
			Scope: gs.Singleton,
		},
		{
			ID:    "user",
			Name:  bean.EnvironmentName,
			Scope: gs.Singleton,
		},
		{
			ID:      "user",
			Name:    "user",
			Aliases: []string{bean.EnvironmentName},
			Scope:   gs.Singleton,
		},
	} {
		definition.Contracts = []gs.Contract{"string"}
		if err := definition.Validate(); err != nil {
			t.Fatal("invalid test definition", err)
		}
		registration := bean.Registration{
			Definition: definition,
			Create:     environmentTestFactory,
		}
		if err := registration.Validate(); !errors.Is(err, &gs.Error{
			Code: gs.InvalidDefinition,
		}) {
			t.Fatal(definition, err)
		}
	}
	if _, err := app.NewApplicationContext(context.Background(), app.WithCatalog(catalog.New()), app.WithDecorator(bean.Decorator{
		Contract: bean.EnvironmentContract,
		Wrap:     environmentTestDecorator,
	})); !errors.Is(err, &gs.Error{
		Code: gs.InvalidDefinition,
	}) {
		t.Fatal(err)
	}
}

func environmentTestFactory(context.Context, bean.Ports) (any, error) {
	return "value", nil
}

func environmentTestDecorator(_ context.Context, value any) (any, error) {
	return value, nil
}

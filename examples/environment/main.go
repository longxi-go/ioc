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

	gosrenv "github.com/longxi-go/gosr/pkg/environment"
	"github.com/longxi-go/ioc"
	app "github.com/longxi-go/ioc/pkg/context"
	nemoenv "github.com/longxi-go/nemo/pkg/environment"
)

// Repository carries configuration read after Environment bootstrap.
type Repository struct {
	Mode string
}

// +ioc:bean:name=repository,aliases={readRepository}
// +ioc:qualifier=memory
// +ioc:profile={dev,test}
// +ioc:conditional:on-property:prefix=company.order.storage,name=mode,havingValue="${expected.mode:memory}"
func NewRepository(environment gosrenv.Environment) (*Repository, error) {
	property, present, err := environment.Lookup("company.order.storage.mode")
	if err != nil {
		return nil, err
	}
	mode, scalar := property.Value.Scalar()
	if !present || !scalar {
		return nil, errors.New("storage mode must be a scalar")
	}
	return &Repository{
		Mode: mode,
	}, nil
}

// Service uses both infrastructure and a qualified application dependency.
// +ioc:component
type Service struct {
	// +ioc:inject
	environment gosrenv.Environment
	// +ioc:inject:name=readRepository,qualifiers={memory}
	repository *Repository
}

func run(ctx context.Context) (result error) {
	application, err := ioc.NewApplicationContext(ctx,
		app.WithEnvironmentOptions(nemoenv.WithProfiles("dev")),
		app.WithProperties(map[string]string{
			"region":                     "dev",
			"storage.dev.mode":           "memory",
			"company.order.storage.mode": "${storage.${region}.mode}",
		}),
	)
	if err != nil {
		return err
	}
	defer func() {
		result = errors.Join(result, application.Close(context.WithoutCancel(ctx)))
	}()
	if err := application.Start(ctx); err != nil {
		return err
	}
	service, err := application.GetBean[*Service](ctx)
	if err != nil {
		return err
	}
	environment, err := application.GetBean[gosrenv.Environment](ctx)
	if err != nil {
		return err
	}
	fmt.Println(service.repository.Mode, service.environment == environment && environment == application.Environment())
	return nil
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

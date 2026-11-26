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

package service

import (
	"context"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/examples/hello/contracts"
)

// Request is cached only within an explicit request lifetime.
// +ioc:component
// +ioc:scope=request
type Request struct {
	ID string
}

// Init demonstrates preparation of an instance created after application startup.
func (r *Request) Init(context.Context) error {
	r.ID = "request-1"
	return nil
}

// Greeting owns no request instance; its provider resolves the caller's lifetime.
// +ioc:component
// +ioc:expose=contracts.Greeter
type Greeting struct {
	// +ioc:inject
	repository contracts.Repository
	// +ioc:inject
	request gs.Provider[*Request]
}

// Greet passes its caller context through the deferred dependency boundary.
func (g *Greeting) Greet(ctx context.Context) (string, error) {
	r, err := g.request(ctx)
	if err != nil {
		return "", err
	}
	return g.repository.Message() + " (" + r.ID + ")", nil
}

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

package context

import (
	"context"
	"errors"

	env "github.com/longxi-go/gosr/pkg/environment"
	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/nemo"
	ne "github.com/longxi-go/nemo/pkg/environment"
	"github.com/longxi-go/nemo/pkg/source"
)

const (
	maximumBuiltinEnvironmentOptions           = 3
	cannotPrepareApplicationEnvironmentMessage = "cannot prepare application Environment"
	conflictingEnvironmentOptionsMessage       = "custom Environment conflicts with built-in construction options or is nil"
	environmentBootstrapPanicMessage           = "environment bootstrap panic"

	propertySourceName = "ioc.properties"
	processSourceName  = "ioc.process"
)

func (c environmentConfiguration) prepare(ctx context.Context) (owner env.Environment, snapshot env.Snapshot, err error) {
	// User-supplied implementations can be typed nil without being nil
	// interfaces. Contain their method panic without reflection or secret text.
	defer func() {
		if p := recover(); p != nil {
			owner = nil
			snapshot = nil
			err = environmentFailure(bootstrapPanicCause(p))
		}
	}()
	if c.supplied {
		if c.owner == nil || c.optionsSupplied || c.propertiesSupplied {
			return nil, nil, &gs.Error{
				Code:    gs.InvalidDefinition,
				Message: conflictingEnvironmentOptionsMessage,
			}
		}
		owner = c.owner
		snapshot = c.owner.Snapshot()
	} else {
		options := make([]nemo.Option, 0, len(c.options)+maximumBuiltinEnvironmentOptions)
		if c.propertiesSupplied {
			properties, e := source.NewMap(propertySourceName, c.properties)
			if e != nil {
				return nil, nil, environmentFailure(e)
			}
			options = append(options, ne.WithProviders(source.Static(properties)))
		}
		options = append(options, c.options...)
		process, e := newProcessProvider()
		if e != nil {
			return nil, nil, environmentFailure(e)
		}
		options = append(options, ne.WithProviders(process), ne.WithDiscovery(source.Discovery{}))
		prepared, e := nemo.New(ctx, options...)
		if e != nil {
			return nil, nil, environmentFailure(e)
		}
		owner = prepared
		snapshot = prepared.Snapshot()
	}
	if snapshot == nil {
		return nil, nil, environmentFailure(nil)
	}
	if _, e := snapshot.AcceptsProfiles(); e != nil {
		return nil, nil, environmentFailure(e)
	}
	if e := ctx.Err(); e != nil {
		return nil, nil, environmentFailure(e)
	}
	return owner, snapshot, nil
}

func environmentFailure(cause error) error {
	return &env.Error{
		Code:    env.InvalidState,
		Message: cannotPrepareApplicationEnvironmentMessage,
		Cause:   cause,
	}
}

func bootstrapPanicCause(value any) error {
	if cause, ok := value.(error); ok {
		return cause
	}
	return errors.New(environmentBootstrapPanicMessage)
}

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

package container

import (
	"context"
	"fmt"
	"slices"

	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	callbackPanicCauseFormat                        = "callback panic: %w"
	callbackPanicValueFormat                        = "callback panic: %v"
	constructedValueViolatesExposedContractsMessage = "constructed value violates exposed contracts"
	decoratorFailedMessage                          = "decorator failed"
	decoratorReturnedNoObjectMessage                = "decorator returned no object"
	factoryFailedMessage                            = "factory failed"
	factoryReturnedNoObjectMessage                  = "factory returned no object"
	initializationFailedMessage                     = "initialization failed"
)

func invoke(fn operationFunc) (err error) {
	defer func() {
		if p := recover(); p != nil {
			if cause, ok := p.(error); ok {
				err = fmt.Errorf(callbackPanicCauseFormat, cause)
			} else {
				err = fmt.Errorf(callbackPanicValueFormat, p)
			}
		}
	}()
	return fn()
}

func (e *Engine) create(ctx context.Context, t *task, b *bound, o *owner) (any, *allocation, error) {
	if err := e.initializePrerequisites(ctx, b, o); err != nil {
		return nil, nil, err
	}
	r := b.recipe
	var value any
	err := invoke(func() error {
		var err error
		value, err = r.Create(ctx, ports{
			e,
			b,
		})
		return err
	})
	var record *allocation
	if value != nil {
		record = &allocation{
			recipe: r,
			value:  value,
		}
		if o != nil &&
			(r.Stop != nil || r.Destroy != nil || r.Start != nil) {
			e.mu.Lock()
			o.records = append(o.records, record)
			e.mu.Unlock()
		}
	}
	if err != nil {
		return value, record, recipeFailure(r, gs.CreationFailed, factoryFailedMessage, err)
	}
	if value == nil {
		return nil, nil, recipeFailure(r, gs.CreationFailed, factoryReturnedNoObjectMessage, nil)
	}
	for _, d := range e.opts.Decorators {
		if slices.Contains(r.Definition.Contracts, d.Contract) {
			var wrapped any
			err = invoke(func() error {
				var err error
				wrapped, err = d.Wrap(ctx, value)
				return err
			})
			if err != nil {
				return value, record, recipeFailure(r, gs.CreationFailed, decoratorFailedMessage, err)
			}
			if wrapped == nil {
				return value, record, recipeFailure(r, gs.ContractMismatch, decoratorReturnedNoObjectMessage, nil)
			}
			value = wrapped
		}
	}
	if r.Accept != nil {
		accepted := false
		err = invoke(func() error {
			accepted = r.Accept(value)
			return nil
		})
		if err != nil || !accepted {
			return value, record, recipeFailure(r, gs.ContractMismatch, constructedValueViolatesExposedContractsMessage, err)
		}
	}
	// Invalid replacement/contract callbacks must compensate the constructor's
	// original allocation. Ownership transfers only after replacement validation.
	record.value = value
	if r.Init != nil {
		err = invoke(func() error {
			return r.Init(ctx, value)
		})
		if err != nil {
			return value, record, recipeFailure(r, gs.CreationFailed, initializationFailedMessage, err)
		}
	}
	if err = ctx.Err(); err != nil {
		return value, record, err
	}
	return value, record, nil
}

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
	"strings"

	env "github.com/longxi-go/gosr/pkg/environment"
	"github.com/longxi-go/nemo/pkg/source"
)

// processProvider preserves process text as data while retaining the standard
// OS provider's normalization, provenance, validation and cancellation behavior.
type processProvider struct {
	source.Provider
}

func newProcessProvider() (source.Provider, error) {
	provider, err := source.NewOS(processSourceName, "")
	if err != nil {
		return nil, err
	}
	return processProvider{
		Provider: provider,
	}, nil
}

func (p processProvider) Fetch(ctx context.Context) (env.PropertySource, error) {
	properties, err := p.Provider.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return processProperties{
		PropertySource: properties,
	}, nil
}

// processProperties escapes only the resolver grammar boundary. Embedding keeps
// source naming and detached key enumeration identical to the OS source.
type processProperties struct {
	env.PropertySource
}

func (p processProperties) Lookup(key string) (env.Property, bool, error) {
	property, found, err := p.PropertySource.Lookup(key)
	if err != nil || !found {
		return property, found, err
	}
	if property.Value.Kind() == env.StringKind {
		text, _ := property.Value.Scalar()
		property.Value = env.String(strings.ReplaceAll(text, env.PlaceholderPrefix, env.EscapedPlaceholderPrefix))
	}
	return property, found, nil
}

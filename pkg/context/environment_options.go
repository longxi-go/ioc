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
	"slices"

	env "github.com/longxi-go/gosr/pkg/environment"
	"github.com/longxi-go/nemo"
	"github.com/longxi-go/nemo/pkg/maptypes"
)

type environmentConfiguration struct {
	owner              SnapshotEnvironment
	supplied           bool
	options            []nemo.Option
	optionsSupplied    bool
	properties         maptypes.ValueMap
	propertiesSupplied bool
}

// WithEnvironment uses an already prepared owner. The application does not
// start its watchers or assume responsibility for external resource shutdown.
// It cannot be combined with the options that construct the built-in Nemo.
func WithEnvironment(owner SnapshotEnvironment) Option {
	return func(c *configuration) {
		c.environment.owner = owner
		c.environment.supplied = true
	}
}

// WithEnvironmentOptions adds Nemo construction policies before initial loading.
// Explicit providers precede the process source and discovered files. Discovery
// options are additive to the optional application.yml default.
func WithEnvironmentOptions(options ...nemo.Option) Option {
	options = slices.Clone(options)
	return func(c *configuration) {
		c.environment.options = append(c.environment.options, options...)
		c.environment.optionsSupplied = true
	}
}

// WithProperties copies programmatic strings into the highest-priority source.
// Conditions resolve them through Environment just like other property sources.
func WithProperties(properties map[string]string) Option {
	copied := make(maptypes.ValueMap, len(properties))
	for key, value := range properties {
		copied[key] = env.String(value)
	}
	return func(c *configuration) {
		c.environment.properties = copied
		c.environment.propertiesSupplied = true
	}
}

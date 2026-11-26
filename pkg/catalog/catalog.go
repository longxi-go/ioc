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

package catalog

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	gs "github.com/longxi-go/gosr/pkg/ioc"
	"github.com/longxi-go/ioc/pkg/bean"
)

const (
	emptyPlanIdentityMessage                       = "empty plan identity"
	incompatibleBatchSchemaMessage                 = "incompatible batch schema; regenerate adapters with the matching ioctl and runtime"
	invalidPlanClosureMessage                      = "invalid plan closure"
	missingBatchIdentityMessage                    = "missing batch identity"
	multipleApplicationPlansRequireWithPlanMessage = "multiple application plans require WithPlan"
	unknownPlanFormat                              = "unknown plan %q"
)

// Catalog is safe for concurrent contribution and snapshot. Invalid contributions
// remain observable through Snapshot; no partially valid batch is published.
type Catalog struct {
	mu      sync.RWMutex
	batches map[string]bool
	recipes map[string]bean.Registration
	plans   map[string]bean.Plan
	errs    []error
}

// New creates an independent definition-only catalog.
func New() *Catalog {
	return &Catalog{
		batches: map[string]bool{},
		recipes: map[string]bean.Registration{},
		plans:   map[string]bean.Plan{},
	}
}

var defaultCatalog = New()

// Default is the process-wide generated contribution destination.
func Default() *Catalog {
	return defaultCatalog
}

// Contribute validates an entire batch before publishing copied recipes.
func (c *Catalog) Contribute(b bean.Batch) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.validate(b)
	if err != nil {
		c.errs = append(c.errs, err)
		return
	}
	for _, r := range b.Registrations {
		c.recipes[r.Definition.ID] = r.Clone()
	}
	c.batches[b.ID] = true
}

func (c *Catalog) validate(b bean.Batch) error {
	if b.ID == "" {
		return &gs.Error{
			Code:    gs.InvalidDefinition,
			Message: missingBatchIdentityMessage,
		}
	}
	if b.Schema != bean.SchemaVersion {
		return &gs.Error{
			Code:    gs.InvalidDefinition,
			Message: incompatibleBatchSchemaMessage,
		}
	}
	if c.batches[b.ID] {
		return &gs.Error{
			Code: gs.DuplicateDefinition,
			Bean: b.ID,
		}
	}
	seen := map[string]bool{}
	for _, r := range b.Registrations {
		if err := r.Validate(); err != nil {
			return err
		}
		id := r.Definition.ID
		if seen[id] {
			return &gs.Error{
				Code: gs.DuplicateDefinition,
				Bean: id,
			}
		}
		if _, ok := c.recipes[id]; ok {
			return &gs.Error{
				Code: gs.DuplicateDefinition,
				Bean: id,
			}
		}
		seen[id] = true
	}
	return nil
}

// Install records an application-owned query and discovery closure.
func (c *Catalog) Install(p bean.Plan) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p.ID == "" {
		c.errs = append(c.errs, &gs.Error{
			Code:    gs.InvalidDefinition,
			Message: emptyPlanIdentityMessage,
		})
		return
	}
	if _, ok := c.plans[p.ID]; ok {
		c.errs = append(c.errs, &gs.Error{
			Code: gs.DuplicateDefinition,
			Bean: p.ID,
		})
		return
	}
	p.DefinitionIDs = slices.Clone(p.DefinitionIDs)
	c.plans[p.ID] = p
}

// Snapshot returns independent metadata and reports all recorded contribution errors.
// Without a selected plan all contributed recipes are included for manual use.
func (c *Catalog) Snapshot(planID string) (bean.Plan, []bean.Registration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.errs) > 0 {
		return bean.Plan{}, nil, errors.Join(c.errs...)
	}
	var p bean.Plan
	if planID != "" {
		var ok bool
		p, ok = c.plans[planID]
		if !ok {
			return p, nil, &gs.Error{
				Code:    gs.InvalidDefinition,
				Message: fmt.Sprintf(unknownPlanFormat, planID),
			}
		}
	} else if len(c.plans) == 1 {
		for _, v := range c.plans {
			p = v
		}
	} else if len(c.plans) > 1 {
		return p, nil, &gs.Error{
			Code:    gs.InvalidDefinition,
			Message: multipleApplicationPlansRequireWithPlanMessage,
		}
	}
	ids := slices.Clone(p.DefinitionIDs)
	if p.ID == "" {
		for id := range c.recipes {
			ids = append(ids, id)
		}
		slices.Sort(ids)
	}
	seen := map[string]bool{}
	rs := make([]bean.Registration, 0, len(ids))
	for _, id := range ids {
		r, ok := c.recipes[id]
		if !ok || seen[id] {
			return p, nil, &gs.Error{
				Code:    gs.InvalidDefinition,
				Bean:    id,
				Message: invalidPlanClosureMessage,
			}
		}
		seen[id] = true
		rs = append(rs, r.Clone())
	}
	p.DefinitionIDs = slices.Clone(p.DefinitionIDs)
	return p, rs, nil
}

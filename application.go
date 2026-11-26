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

package ioc

import (
	"context"

	managed "github.com/longxi-go/ioc/pkg/context"
)

// NewApplicationContext delegates composition to the public application package.
// It snapshots definitions; callers explicitly Start and Close their application.
func NewApplicationContext(ctx context.Context, options ...managed.Option) (*managed.Application, error) {
	return managed.NewApplicationContext(ctx, options...)
}

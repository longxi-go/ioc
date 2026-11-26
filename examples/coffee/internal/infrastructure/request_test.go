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

package infrastructure

import (
	"context"
	"testing"
)

func TestOperationMetadataAndCancellation(t *testing.T) {
	if _, err := NewOperation(context.Background()); err == nil {
		t.Fatal("missing metadata accepted")
	}
	operation, err := NewOperation(WithOperation(context.Background(), "request-1"))
	if err != nil || operation.ID != "request-1" {
		t.Fatal(operation, err)
	}
	ctx, cancel := context.WithCancel(WithOperation(context.Background(), "cancelled"))
	cancel()
	if _, err := NewOperation(ctx); err == nil {
		t.Fatal("cancellation ignored")
	}
}

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

package bean_test

import (
	"context"
	"testing"

	"github.com/longxi-go/ioc/pkg/bean"
)

type applicationPredicate func(any) bool

type applicationRelease func(context.Context) error

func acceptApplicationValue(value any) bool {
	return value != nil
}

func releaseApplicationValue(ctx context.Context) error {
	return ctx.Err()
}

func TestCallbackAliasesCompatibility(t *testing.T) {
	var original applicationPredicate = acceptApplicationValue
	var accept bean.AcceptFunc = original
	original = accept
	registration := bean.Registration{
		Accept: original,
	}
	if !registration.Accept("typed") || registration.Accept(nil) {
		t.Fatal("application predicate lost")
	}
	var releaseOriginal applicationRelease = releaseApplicationValue
	var release bean.ReleaseFunc = releaseOriginal
	releaseOriginal = release
	if err := releaseOriginal(context.Background()); err != nil {
		t.Fatal(err)
	}
}

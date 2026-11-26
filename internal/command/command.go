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

package command

import (
	"context"
	"io"

	"github.com/longxi-go/ioc/pkg/cli"
)

// Run retains ioctl's entry point while public cli owns parsing and reporting.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return cli.Run(ctx, args, stdout, stderr, cli.Options{})
}

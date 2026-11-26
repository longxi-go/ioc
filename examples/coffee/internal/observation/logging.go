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

package observation

import (
	"log/slog"
	"os"
)

// NewApplicationLogger is the primary logger; qualified loggers remain selectable.
// +ioc:bean:name=applicationLogger,aliases={defaultLogger}
// +ioc:primary
func NewApplicationLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// NewAuditLogger provides a separately selectable audit channel.
// +ioc:bean:name=auditLogger
// +ioc:qualifier=audit
func NewAuditLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil)).With("channel", "audit")
}

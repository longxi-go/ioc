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

package configuration

import (
	"errors"
	"net"
	"strconv"
	"strings"

	gosrenv "github.com/longxi-go/gosr/pkg/environment"
	"github.com/longxi-go/nemo/pkg/binding"
)

const (
	AddressKey             = "cafe.server.address"
	DatabaseKey            = "cafe.database.path"
	ReceiptKey             = "cafe.features.receipt.enabled"
	invalidSettingsMessage = "invalid cafe settings"
)

// Settings is bound once after Environment bootstrap and before opening resources.
// +nemo:config:prefix=cafe
type Settings struct {
	Server   Server
	Database Database
	Features Features
}

type Server struct {
	Address string `default:"127.0.0.1:8082"`
}

type Database struct {
	Path string `default:"coffee.db"`
}

type Features struct {
	Receipt Receipt
}

type Receipt struct {
	Enabled bool `default:"false"`
}

// NewSettings uses the same prepared Environment that selects conditional Beans.
// +ioc:bean
func NewSettings(env gosrenv.Environment) (*Settings, error) {
	settings, err := binding.Bind[Settings](env, "")
	if err != nil {
		return nil, err
	}
	if err = settings.Validate(); err != nil {
		return nil, err
	}
	return &settings, nil
}

// Validate prevents starting HTTP or storage with incomplete settings.
func (s Settings) Validate() error {
	_, port, err := net.SplitHostPort(s.Server.Address)
	number, parseErr := strconv.Atoi(port)
	if err != nil || parseErr != nil || number < 1 || number > 65535 || strings.TrimSpace(s.Database.Path) == "" {
		return errors.New(invalidSettingsMessage)
	}
	return nil
}

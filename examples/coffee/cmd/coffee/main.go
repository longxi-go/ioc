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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/longxi-go/ioc/examples/coffee/internal/application"
	"github.com/longxi-go/ioc/examples/coffee/internal/configuration"
	managed "github.com/longxi-go/ioc/pkg/context"
	nemoenv "github.com/longxi-go/nemo/pkg/environment"
)

const (
	databaseFlag = "db"
	addressFlag  = "address"
	profileFlag  = "profile"
)

func main() {
	database := flag.String(databaseFlag, "", "SQLite file path (overrides configuration)")
	address := flag.String(addressFlag, "", "HTTP listen address (overrides configuration)")
	profile := flag.String(profileFlag, "", "Active configuration profile, for example dev")
	flag.Parse()
	properties := map[string]string{}
	if *database != "" {
		properties[configuration.DatabaseKey] = *database
	}
	if *address != "" {
		properties[configuration.AddressKey] = *address
	}
	options := []managed.Option{managed.WithProperties(properties)}
	if *profile != "" {
		options = append(options, managed.WithEnvironmentOptions(nemoenv.WithProfiles(*profile)))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := application.Run(ctx, options...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

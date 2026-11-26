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

package application

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	managed "github.com/longxi-go/ioc/pkg/context"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 32 << 10
)

// NewServer applies connection timeouts and header limits before HTTP admission.
func NewServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

// Run attempts bounded HTTP drain before closing the graph and joins shutdown errors.
func Run(ctx context.Context, options ...managed.Option) (result error) {
	app, err := Start(ctx, options...)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := cleanupContext(ctx)
		defer cancel()
		result = errors.Join(result, app.Close(cleanup))
	}()
	server := NewServer(app.settings.Server.Address, app.Router())
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	return serve(ctx, server, listener)
}

// serve owns the listener while draining requests before its caller closes Beans.
func serve(ctx context.Context, server *http.Server, listener net.Listener) error {
	stopped := make(chan error, 1)
	go func() {
		stopped <- server.Serve(listener)
	}()
	var serveErr error
	serveReturned := false
	select {
	case serveErr = <-stopped:
		serveReturned = true
	case <-ctx.Done():
	}
	// Serve can return an accept error while previously accepted requests still run.
	cleanup, cancel := cleanupContext(ctx)
	defer cancel()
	shutdownErr := server.Shutdown(cleanup)
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, server.Close())
	}
	if !serveReturned {
		serveErr = <-stopped
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutdownErr)
}

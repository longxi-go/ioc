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
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestServerDrainsActiveRequestOnCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	clientDone := make(chan error, 1)
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, err := io.WriteString(writer, "finished")
		if err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	server := NewServer(listener.Addr().String(), handler)
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- serve(ctx, server, listener)
	}()
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_, readErr := io.Copy(io.Discard, response.Body)
			err = errors.Join(readErr, response.Body.Close())
		}
		clientDone <- err
	}()
	<-entered
	cancel()
	select {
	case err := <-serverDone:
		t.Fatalf("returned before active request: %v", err)
	default:
	}
	close(release)
	if err := <-clientDone; err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

type failingListener struct {
	net.Listener
	accepted bool
	fail     chan struct{}
	failure  error
}

func (l *failingListener) Accept() (net.Conn, error) {
	if l.accepted {
		<-l.fail
		return nil, l.failure
	}
	connection, err := l.Listener.Accept()
	l.accepted = true
	return connection, err
}

func TestAcceptFailureDrainsActiveRequest(t *testing.T) {
	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("accept failed")
	listener := &failingListener{
		Listener: base,
		fail:     make(chan struct{}),
		failure:  failure,
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	draining := make(chan struct{})
	var once sync.Once
	unblock := func() {
		once.Do(func() {
			close(release)
		})
	}
	defer unblock()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, err := io.WriteString(writer, "finished")
		if err != nil {
			t.Error(err)
		}
	})
	server := NewServer(listener.Addr().String(), handler)
	server.RegisterOnShutdown(func() {
		close(draining)
	})
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	serverDone := make(chan error, 1)
	clientDone := make(chan error, 1)
	go func() {
		serverDone <- serve(ctx, server, listener)
	}()
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_, readErr := io.Copy(io.Discard, response.Body)
			err = errors.Join(readErr, response.Body.Close())
		}
		clientDone <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	close(listener.fail)
	select {
	case err := <-serverDone:
		t.Fatalf("returned while handler active: %v", err)
	case <-draining:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	unblock()
	if err := <-clientDone; err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; !errors.Is(err, failure) {
		t.Fatal("lost accept failure", err)
	}
}

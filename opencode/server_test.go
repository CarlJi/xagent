/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package opencode

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/goplus/xagent"
)

type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (nopWriteCloser) Close() error                { return nil }

type blockingExecutor struct {
	mu    sync.Mutex
	calls int
}

func (e *blockingExecutor) Exec(ctx context.Context, args []string, env map[string]string, workDir string) (*xagent.Process, error) {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()

	return &xagent.Process{
		Stdout: io.NopCloser(bytes.NewReader(nil)),
		Stdin:  nopWriteCloser{},
		Stderr: &bytes.Buffer{},
		Wait: func() (int, error) {
			<-ctx.Done()
			return 0, nil
		},
	}, nil
}

func (e *blockingExecutor) IsHealthy(ctx context.Context) bool { return true }
func (e *blockingExecutor) Close(ctx context.Context) error    { return nil }

func (e *blockingExecutor) Calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func TestServerEnsureStartedIsAsync(t *testing.T) {
	exec := &blockingExecutor{}
	a := New(WithExecutor(exec), WithBinaryPath("/bin/echo"))
	s := &server{agent: a}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ensureStarted(context.Background(), xagent.SessionConfig{})
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ensureStarted error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("ensureStarted blocked on long-running serve process")
	}

	if exec.Calls() != 1 {
		t.Fatalf("expected 1 serve exec call, got %d", exec.Calls())
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.close(closeCtx); err != nil {
		t.Fatalf("close error: %v", err)
	}
}

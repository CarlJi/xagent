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
	"context"
	"io"
	"sync"

	"github.com/goplus/xagent"
)

type server struct {
	agent   *Agent
	started bool
	cancel  context.CancelFunc
	done    chan struct{}
	mu      sync.Mutex
}

func (s *server) ensureStarted(ctx context.Context, cfg xagent.SessionConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	serveCtx, cancel := context.WithCancel(context.Background())
	proc, err := s.agent.exec.Exec(serveCtx, []string{s.agent.command(), "serve", "--port", "4096"}, s.agent.baseEnv(cfg.Env), cfg.WorkDir)
	if err != nil {
		cancel()
		return err
	}
	if proc.Stdin != nil {
		_ = proc.Stdin.Close()
	}
	if proc.Stdout != nil {
		go func() {
			if _, err := io.Copy(io.Discard, proc.Stdout); err != nil {
				// Log error for diagnostics, but don't fail since this is background output draining
				// Consider using a proper logger in production: log.Printf("opencode server stdout drain error: %v", err)
				_ = err // Explicitly acknowledge the error
			}
			if err := proc.Stdout.Close(); err != nil {
				// Log close error if needed: log.Printf("opencode server stdout close error: %v", err)
				_ = err
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = proc.Wait()
	}()
	s.started = true
	s.cancel = cancel
	s.done = done
	return nil
}

func (s *server) close(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	done := s.done
	s.started = false
	s.cancel = nil
	s.done = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

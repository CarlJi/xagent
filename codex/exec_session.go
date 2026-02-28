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

package codex

import (
	"context"
	"sync"
	"time"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/internal/ndjson"
)

type execSession struct {
	agent    *Agent
	cfg      xagent.SessionConfig
	threadID string
	mu       sync.Mutex
	closed   bool
}

func (s *execSession) ID() string { s.mu.Lock(); defer s.mu.Unlock(); return s.threadID }

func (s *execSession) Send(ctx context.Context, prompt string) (xagent.Stream, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, &xagent.AgentError{Code: xagent.ErrInvalidConfig, Agent: "codex", Message: "session closed"}
	}
	threadID := s.threadID
	cfg := s.cfg
	s.mu.Unlock()

	fullArgs := append([]string{s.agent.command()}, buildExecArgs(cfg, prompt, threadID)...)
	proc, err := s.agent.exec.Exec(ctx, fullArgs, s.agent.baseEnv(cfg.Env), cfg.WorkDir)
	if err != nil {
		return nil, err
	}
	if proc.Stdin != nil {
		proc.Stdin.Close()
		proc.Stdin = nil
	}

	scanner := ndjson.NewScanner(proc.Stdout)
	return xagent.NewProcessStream(proc, scanner, parseJSONLLine,
		xagent.WithOnInit(func(init xagent.InitEvent) {
			if init.SessionID != "" {
				s.mu.Lock()
				s.threadID = init.SessionID
				s.mu.Unlock()
			}
		}),
		xagent.WithExitHandler(func(exitCode int, stderr []byte) []xagent.Event {
			return []xagent.Event{xagent.ErrorEvent{Message: string(stderr), Code: xagent.ErrExecution, Fatal: true, Timestamp: time.Now()}}
		}),
	), nil
}

func (s *execSession) IsHealthy(ctx context.Context) bool { return s.agent.exec.IsHealthy(ctx) }
func (s *execSession) Close(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}

var _ xagent.Session = (*execSession)(nil)

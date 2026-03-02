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

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/codex/internal/jsonrpc"
)

type appServerSession struct {
	agent    *Agent
	cfg      xagent.SessionConfig
	threadID string
	client   *jsonrpc.Client
	mu       sync.Mutex
	closed   bool
}

func (s *appServerSession) ID() string { s.mu.Lock(); defer s.mu.Unlock(); return s.threadID }

func (s *appServerSession) Send(ctx context.Context, prompt string) (xagent.Stream, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, &xagent.AgentError{Code: xagent.ErrInvalidConfig, Agent: "codex", Message: "session closed"}
	}
	if s.client == nil {
		s.client = jsonrpc.NewClient(nil)
	}
	client := s.client
	s.mu.Unlock()

	var resp map[string]any
	if err := client.Request(ctx, "conversation.send", map[string]any{"prompt": prompt, "thread_id": s.ID()}, &resp); err != nil {
		return nil, err
	}

	if tid, _ := resp["thread_id"].(string); tid != "" {
		s.mu.Lock()
		s.threadID = tid
		s.mu.Unlock()
	}
	events := []xagent.Event{
		xagent.InitEvent{SessionID: s.ID(), AgentName: "codex"},
		xagent.TextEvent{Delta: str(resp["text"])},
		xagent.TurnCompleteEvent{InputTokens: intNum(resp["input_tokens"]), OutputTokens: intNum(resp["output_tokens"]), StopReason: str(resp["stop_reason"])},
	}
	return xagent.NewSliceStream(events, nil), nil
}

func (s *appServerSession) IsHealthy(ctx context.Context) bool { return s.agent.exec.IsHealthy(ctx) }
func (s *appServerSession) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

var _ xagent.Session = (*appServerSession)(nil)

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

package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/xagenttest"
)

func TestClaudeSendParsesEvents(t *testing.T) {
	fixturePath := filepath.Join("..", "xagenttest", "fixtures", "claude_stream.ndjson")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	exec := xagenttest.NewMockExecutor(
		xagenttest.ExecResponse{Stdout: []byte("claude 2.1.62\n"), ExitCode: 0},
		xagenttest.ExecResponse{Stdout: data, ExitCode: 0},
	)

	a := New(WithExecutor(exec), WithAPIKey("k"))
	ctx := context.Background()
	if err := a.Validate(ctx); err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	sess, err := a.Start(ctx, xagent.SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}

	stream, err := sess.Send(ctx, "hello")
	if err != nil {
		t.Fatal(err)
	}

	events := xagenttest.AssertEvents(t, ctx, stream,
		xagent.EventKindInit,
		xagent.EventKindText,
		xagent.EventKindToolStart,
		xagent.EventKindToolEnd,
		xagent.EventKindThinking,
		xagent.EventKindText,
		xagent.EventKindTurnComplete,
	)
	if sess.ID() == "" {
		t.Fatalf("expected session id after init event")
	}
	init := events[0].(xagent.InitEvent)
	if init.SessionID != "sess-1" {
		t.Errorf("expected session_id=sess-1, got %s", init.SessionID)
	}
	if len(init.ToolNames) != 3 {
		t.Errorf("expected 3 tool names, got %d", len(init.ToolNames))
	}
}

func TestClaudeSendExitCodeProducesErrorEvent(t *testing.T) {
	exec := xagenttest.NewMockExecutor(
		xagenttest.ExecResponse{Stdout: []byte("claude 2.1.62\n"), ExitCode: 0},
		xagenttest.ExecResponse{Stdout: []byte(""), Stderr: []byte("boom"), ExitCode: 1},
	)

	a := New(WithExecutor(exec), WithAPIKey("k"))
	ctx := context.Background()
	if err := a.Validate(ctx); err != nil {
		t.Fatal(err)
	}
	sess, err := a.Start(ctx, xagent.SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := sess.Send(ctx, "hello")
	if err != nil {
		t.Fatal(err)
	}
	events, err := xagenttest.DrainStream(ctx, stream)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if ev, ok := e.(xagent.ErrorEvent); ok && ev.Fatal {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fatal ErrorEvent")
	}
}

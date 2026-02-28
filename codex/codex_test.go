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
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/xagenttest"
)

func TestExecSessionParsesFixture(t *testing.T) {
	fx, err := os.ReadFile(filepath.Join("..", "xagenttest", "fixtures", "codex_exec.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	mock := xagenttest.NewMockExecutor(
		xagenttest.ExecResponse{Stdout: []byte("codex 0.106.0\n")},
		xagenttest.ExecResponse{Stdout: fx},
	)
	a := New(WithExecutor(mock), WithAPIKey("k"), WithBinaryPath("/bin/echo"))
	ctx := context.Background()
	_ = a.Validate(ctx)
	s, err := a.Start(ctx, xagent.SessionConfig{MaxTurns: 1})
	if err != nil {
		t.Fatal(err)
	}
	r, err := xagent.CollectResult(ctx, must(s.Send(ctx, "hi")))
	if err != nil {
		t.Fatal(err)
	}
	if r.Text == "" {
		t.Fatalf("expected text")
	}
}

func must(s xagent.Stream, err error) xagent.Stream {
	if err != nil {
		panic(err)
	}
	return s
}

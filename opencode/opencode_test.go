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
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/xagenttest"
)

func TestOpenCodeParsesFixture(t *testing.T) {
	fx, err := os.ReadFile(filepath.Join("..", "xagenttest", "fixtures", "opencode_sse.txt"))
	if err != nil {
		t.Fatal(err)
	}
	mock := xagenttest.NewMockExecutor(
		xagenttest.ExecResponse{Stdout: []byte("opencode 1.2.15\n")},
		xagenttest.ExecResponse{Stdout: []byte("ok")},
		xagenttest.ExecResponse{Stdout: fx},
	)
	a := New(WithExecutor(mock), WithBinaryPath("/bin/echo"))
	ctx := context.Background()
	_ = a.Validate(ctx)
	s, err := a.Start(ctx, xagent.SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := xagent.CollectResult(ctx, must(s.Send(ctx, "hello")))
	if err != nil {
		t.Fatal(err)
	}
	if res.Text == "" {
		t.Fatalf("expected text")
	}
}

func must(s xagent.Stream, err error) xagent.Stream {
	if err != nil {
		panic(err)
	}
	return s
}

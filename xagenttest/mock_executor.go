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

package xagenttest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/goplus/xagent"
)

// ExecResponse configures a single mock execution result.
type ExecResponse struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
}

// MockExecCall records the arguments of an Exec() invocation.
type MockExecCall struct {
	Args    []string
	Env     map[string]string
	WorkDir string
}

// MockExecutor returns pre-configured responses for each Exec() call.
type MockExecutor struct {
	mu        sync.Mutex
	responses []ExecResponse
	calls     []MockExecCall
}

// NewMockExecutor creates a MockExecutor with a queue of responses.
func NewMockExecutor(responses ...ExecResponse) *MockExecutor {
	return &MockExecutor{responses: responses}
}

func (m *MockExecutor) Exec(ctx context.Context, args []string, env map[string]string, workDir string) (*xagent.Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, MockExecCall{Args: args, Env: env, WorkDir: workDir})
	if len(m.responses) == 0 {
		return nil, fmt.Errorf("mock executor: no response configured")
	}
	resp := m.responses[0]
	m.responses = m.responses[1:]
	if resp.Err != nil {
		return nil, resp.Err
	}

	stderrBuf := &bytes.Buffer{}
	stderrBuf.Write(resp.Stderr)

	return &xagent.Process{
		Stdout: io.NopCloser(bytes.NewReader(resp.Stdout)),
		Stdin:  nopWriteCloser{},
		Stderr: stderrBuf,
		Wait: func() (int, error) {
			return resp.ExitCode, nil
		},
	}, nil
}

func (m *MockExecutor) IsHealthy(ctx context.Context) bool { return true }
func (m *MockExecutor) Close(ctx context.Context) error    { return nil }

// Calls returns a copy of all recorded Exec() invocations.
func (m *MockExecutor) Calls() []MockExecCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MockExecCall, len(m.calls))
	copy(out, m.calls)
	return out
}

// nopWriteCloser is an io.WriteCloser that discards writes.
type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (nopWriteCloser) Close() error                { return nil }

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

package xagent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// LocalExecutor executes commands on the local machine using os/exec.
type LocalExecutor struct{}

// NewLocalExecutor returns a new LocalExecutor.
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

// Exec starts a local subprocess and returns a Process handle for streaming I/O.
func (e *LocalExecutor) Exec(ctx context.Context, args []string, env map[string]string, workDir string) (*Process, error) {
	if len(args) == 0 {
		return nil, &AgentError{Code: ErrInvalidConfig, Agent: "local", Message: "empty args"}
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Build environment: start from os.Environ(), filter CLAUDECODE*, then merge explicit env.
	baseEnv := filterClaudeCodeEnv(os.Environ())
	for k, v := range env {
		baseEnv = append(baseEnv, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = baseEnv

	// Set process attributes for clean cleanup (platform-specific).
	setProcessAttrs(cmd)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &Process{
		Stdout: stdoutPipe,
		Stdin:  stdinPipe,
		Stderr: &stderrBuf,
		Wait: func() (int, error) {
			err := cmd.Wait()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return exitErr.ExitCode(), nil
				}
				return 0, err
			}
			return 0, nil
		},
	}, nil
}

// filterClaudeCodeEnv removes CLAUDECODE and CLAUDECODE_SESSION from the
// environment slice, preventing nested Claude Code sessions from interfering.
func filterClaudeCodeEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, "CLAUDECODE=") || strings.HasPrefix(kv, "CLAUDECODE_SESSION=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// IsHealthy reports whether the local executor is usable.
// It returns false only if ctx is non-nil and already cancelled.
func (e *LocalExecutor) IsHealthy(ctx context.Context) bool {
	return ctx == nil || ctx.Err() == nil
}

// Close is a no-op for the local executor; there are no external resources to release.
func (e *LocalExecutor) Close(ctx context.Context) error {
	return nil
}

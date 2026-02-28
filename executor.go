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
	"io"
)

// Executor abstracts command execution. Default is LocalExecutor (os/exec).
// Sandbox integrations (Docker, E2B) implement this interface.
type Executor interface {
	// Exec starts a command and returns a Process handle for streaming I/O.
	// args[0] is the binary name; args[1:] are arguments.
	// env is merged into the process environment.
	// workDir sets the working directory; empty string means inherit.
	Exec(ctx context.Context, args []string, env map[string]string, workDir string) (*Process, error)

	// IsHealthy reports whether the execution environment is usable.
	IsHealthy(ctx context.Context) bool

	// Close releases execution environment resources.
	Close(ctx context.Context) error
}

// Process is the handle returned by Executor.Exec.
type Process struct {
	Stdout io.ReadCloser                    // subprocess stdout (NDJSON/SSE stream)
	Stdin  io.WriteCloser                   // subprocess stdin; nil when not writable
	Wait   func() (exitCode int, err error) // blocks until process exits
	Stderr *bytes.Buffer                    // captured stderr; read after Wait() returns
}

// CommandBuilder converts SessionConfig into executable command components.
type CommandBuilder interface {
	BuildArgs(cfg SessionConfig, prompt string) []string
	BinaryName() string
	EnvVars(cfg SessionConfig) map[string]string
}

// RunCollect is a convenience helper for one-shot commands (e.g. --version).
// It executes the command, collects all stdout, waits for exit, and returns results.
func RunCollect(ctx context.Context, exec Executor, args []string, env map[string]string, workDir string) (stdout []byte, stderr []byte, exitCode int, err error) {
	proc, execErr := exec.Exec(ctx, args, env, workDir)
	if execErr != nil {
		return nil, nil, 0, execErr
	}
	stdoutData, readErr := io.ReadAll(proc.Stdout)
	proc.Stdout.Close()
	if proc.Stdin != nil {
		proc.Stdin.Close()
	}
	exitCode, waitErr := proc.Wait()
	var stderrData []byte
	if proc.Stderr != nil {
		stderrData = proc.Stderr.Bytes()
	}
	if readErr != nil {
		return stdoutData, stderrData, exitCode, readErr
	}
	if waitErr != nil {
		return stdoutData, stderrData, exitCode, waitErr
	}
	return stdoutData, stderrData, exitCode, nil
}

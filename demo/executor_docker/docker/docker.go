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

package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/goplus/xagent"
)

// Mount defines a host-to-container volume bind mount.
type Mount struct {
	Host      string // absolute path on the Docker host
	Container string // absolute path inside the container
}

// Option is a functional option for configuring a docker Executor.
type Option func(*Executor)

// Executor implements xagent.Executor by running commands via `docker exec`
// against a long-lived container managed by the executor's lifecycle.
type Executor struct {
	image         string
	containerName string
	user          string
	workDir       string
	mounts        []Mount
	pathRemap     map[string]string
	init          bool
	autoRemove    bool
	logger        *slog.Logger

	mu      sync.Mutex
	started bool
}

// New creates a new docker Executor and validates that required options are present.
// It returns an error if image or containerName are not set.
func New(opts ...Option) (*Executor, error) {
	e := &Executor{
		pathRemap: map[string]string{},
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
	for _, opt := range opts {
		opt(e)
	}
	if e.image == "" {
		return nil, &xagent.AgentError{Code: xagent.ErrInvalidConfig, Agent: "docker", Message: "image is required"}
	}
	if e.containerName == "" {
		return nil, &xagent.AgentError{Code: xagent.ErrInvalidConfig, Agent: "docker", Message: "container name is required"}
	}
	return e, nil
}

// WithImage sets the Docker image to use when creating the container.
func WithImage(image string) Option {
	return func(e *Executor) { e.image = image }
}

// WithContainerName sets the name of the Docker container.
// A deterministic name allows the executor to reuse an already-running container.
func WithContainerName(name string) Option {
	return func(e *Executor) { e.containerName = name }
}

// WithUser sets the user (name or UID) under which commands are executed inside the container.
func WithUser(user string) Option {
	return func(e *Executor) { e.user = user }
}

// WithWorkDir sets the default working directory inside the container.
func WithWorkDir(dir string) Option {
	return func(e *Executor) { e.workDir = dir }
}

// WithMounts configures bind mounts to pass to `docker run`.
func WithMounts(mounts []Mount) Option {
	return func(e *Executor) { e.mounts = append([]Mount(nil), mounts...) }
}

// WithPathRemap registers a mapping from host paths to in-container paths.
// When a WorkDir matches a host path prefix, it is transparently rewritten
// to the corresponding container path during `docker exec` invocations.
func WithPathRemap(pathRemap map[string]string) Option {
	return func(e *Executor) {
		m := make(map[string]string, len(pathRemap))
		for k, v := range pathRemap {
			m[k] = v
		}
		e.pathRemap = m
	}
}

// WithInit sets the --init flag on `docker run`, ensuring PID 1 runs an init process
// to reap zombie child processes.
func WithInit(v bool) Option {
	return func(e *Executor) { e.init = v }
}

// WithAutoRemove sets the --rm flag on `docker run`. When true, the container is
// automatically deleted by Docker after it stops.
func WithAutoRemove(v bool) Option {
	return func(e *Executor) { e.autoRemove = v }
}

// WithLogger sets a custom structured logger for the executor.
func WithLogger(l *slog.Logger) Option {
	return func(e *Executor) {
		if l != nil {
			e.logger = l
		}
	}
}

func (e *Executor) Exec(ctx context.Context, args []string, env map[string]string, workDir string) (*xagent.Process, error) {
	if err := e.ensureContainer(ctx); err != nil {
		return nil, err
	}

	dockerArgs := []string{"exec", "-i"}
	if e.user != "" {
		dockerArgs = append(dockerArgs, "--user", e.user)
	}
	if dir := e.mapWorkDir(workDir); dir != "" {
		dockerArgs = append(dockerArgs, "-w", dir)
	}
	for k, v := range env {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	dockerArgs = append(dockerArgs, e.containerName)
	dockerArgs = append(dockerArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

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

	return &xagent.Process{
		Stdout: stdoutPipe,
		Stdin:  stdinPipe,
		Stderr: &stderrBuf,
		Wait: func() (int, error) {
			err := cmd.Wait()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					if status, ok2 := exitErr.Sys().(syscall.WaitStatus); ok2 {
						return status.ExitStatus(), nil
					}
					return 1, nil
				}
				return 0, err
			}
			return 0, nil
		},
	}, nil
}

func (e *Executor) IsHealthy(ctx context.Context) bool {
	if err := e.ensureContainer(ctx); err != nil {
		return false
	}
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", e.containerName)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func (e *Executor) Close(ctx context.Context) error {
	e.mu.Lock()
	started := e.started
	e.mu.Unlock()
	if !started {
		return nil
	}
	if e.autoRemove {
		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", e.containerName)
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	}
	cmd := exec.CommandContext(ctx, "docker", "stop", e.containerName)
	return cmd.Run()
}

func (e *Executor) ensureContainer(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.started {
		return nil
	}

	running, exists := e.containerState(ctx)
	if exists && running {
		e.started = true
		return nil
	}
	if exists && !running {
		if err := exec.CommandContext(ctx, "docker", "start", e.containerName).Run(); err != nil {
			return err
		}
		e.started = true
		return nil
	}

	args := []string{"run", "-d", "--name", e.containerName}
	if e.init {
		args = append(args, "--init")
	}
	if e.autoRemove {
		args = append(args, "--rm")
	}
	if e.user != "" {
		args = append(args, "--user", e.user)
	}
	if e.workDir != "" {
		args = append(args, "-w", e.workDir)
	}
	for _, m := range e.mounts {
		host := m.Host
		if host != "" {
			if abs, err := filepath.Abs(host); err == nil {
				host = abs
			}
		}
		args = append(args, "-v", fmt.Sprintf("%s:%s", host, m.Container))
	}
	args = append(args, e.image, "sleep", "infinity")

	if _, _, _, err := runWithOutput(exec.CommandContext(ctx, "docker", args...)); err != nil {
		return err
	}
	e.started = true
	return nil
}

func (e *Executor) containerState(ctx context.Context) (running bool, exists bool) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", e.containerName)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.ExitStatus() != 0 {
				return false, false
			}
		}
		return false, false
	}
	return strings.TrimSpace(string(out)) == "true", true
}

func (e *Executor) mapWorkDir(workDir string) string {
	if workDir == "" {
		return e.workDir
	}
	if e.pathRemap == nil {
		return workDir
	}
	for host, container := range e.pathRemap {
		host = filepath.Clean(host)
		work := filepath.Clean(workDir)
		if work == host {
			return container
		}
		prefix := host + string(os.PathSeparator)
		if rest, ok := strings.CutPrefix(work, prefix); ok {
			return filepath.ToSlash(filepath.Join(container, rest))
		}
	}
	return workDir
}

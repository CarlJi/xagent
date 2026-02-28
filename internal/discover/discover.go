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

package discover

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/goplus/xagent"
)

// Find searches for a binary named name using the same resolution strategy as FindWithPath
// but without an explicit path override.
func Find(name string) (string, error) {
	return FindWithPath(name, "")
}

// FindWithPath locates the named binary by first checking explicitPath (if non-empty),
// then PATH, then a set of common installation directories, and finally the login shell.
// On Windows it appends .cmd and .exe suffixes when appropriate.
// It returns an AgentError with code ErrBinaryNotFound when the binary cannot be located.
func FindWithPath(name, explicitPath string) (string, error) {
	if explicitPath != "" {
		if p, ok := validateExplicit(name, explicitPath); ok {
			return p, nil
		}
		return "", &xagent.AgentError{Code: xagent.ErrBinaryNotFound, Agent: name, Message: "explicit binary path not executable"}
	}

	if p, err := exec.LookPath(name); err == nil {
		return filepath.Abs(p)
	}

	for _, p := range candidatePaths(name) {
		if isExecutable(p) {
			return p, nil
		}
	}

	if p := findFromLoginShell(name); p != "" {
		return p, nil
	}

	return "", &xagent.AgentError{Code: xagent.ErrBinaryNotFound, Agent: name, Message: fmt.Sprintf("binary %q not found", name)}
}

func validateExplicit(name, explicit string) (string, bool) {
	p := explicit
	if !filepath.IsAbs(p) {
		abs, err := filepath.Abs(p)
		if err == nil {
			p = abs
		}
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(p), ".cmd") && !strings.HasSuffix(strings.ToLower(p), ".exe") && !strings.HasSuffix(strings.ToLower(p), ".bat") {
		if isExecutable(p + ".cmd") {
			return p + ".cmd", true
		}
		if isExecutable(p + ".exe") {
			return p + ".exe", true
		}
		if isExecutable(p + ".bat") {
			return p + ".bat", true
		}
	}
	return p, isExecutable(p)
}

func candidatePaths(name string) []string {
	home, homeErr := os.UserHomeDir()
	var out []string
	bases := []string{
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/usr/bin",
	}
	if homeErr == nil && home != "" {
		bases = append([]string{
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".npm", "bin"),
			filepath.Join(home, ".claude", "local"),
		}, bases...)
	}
	for _, base := range bases {
		if base == "" {
			continue
		}
		out = append(out, filepath.Join(base, name))
		if runtime.GOOS == "windows" {
			out = append(out, filepath.Join(base, name+".cmd"))
			out = append(out, filepath.Join(base, name+".exe"))
			out = append(out, filepath.Join(base, name+".bat"))
		}
	}
	return out
}

func findFromLoginShell(name string) string {
	sh := "zsh"
	if _, err := exec.LookPath(sh); err != nil {
		sh = "bash"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, sh, "-lc", "command -v -- \"$1\"", "_", name)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

func isExecutable(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	if st.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".cmd" || ext == ".exe" || ext == ".bat"
	}
	return st.Mode()&0o111 != 0
}

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
	"os/exec"
)

// -----------------------------------------------------------------------------

type Client struct {
	workDir string
}

// New creates a new Claude CLI client with the specified working directory.
func New(workDir string) Client {
	return Client{workDir: workDir}
}

// Name returns the name of the client, which is "claude".
func (c Client) Name() string {
	return "claude"
}

// Login logs in to the Claude CLI using the provided email and SSO flag.
// If the SSO is true, it will use single sign-on for authentication.
func (c Client) Login(ctx context.Context, email string, sso bool) error {
	args := append(make([]string, 0, 5), "auth", "login", "--email", email)
	if sso {
		args = append(args, "--sso")
	}
	cmd := exec.CommandContext(ctx, c.Name(), args...)
	cmd.Dir = c.workDir
	return cmd.Run()
}

// Logout logs out of the Claude CLI.
func (c Client) Logout(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.Name(), "auth", "logout")
	cmd.Dir = c.workDir
	return cmd.Run()
}

// -----------------------------------------------------------------------------

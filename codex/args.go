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

import "github.com/goplus/xagent"

func buildExecArgs(cfg xagent.SessionConfig, prompt string, threadID string) []string {
	if threadID != "" {
		args := []string{"exec", "resume", threadID, "--json"}
		if cfg.Permission == xagent.PermAutoApprove {
			args = append(args, "--full-auto")
		}
		args = append(args, "--", prompt)
		return args
	}
	args := []string{"exec", "--json", "--ephemeral"}
	if cfg.Permission == xagent.PermAutoApprove {
		args = append(args, "--full-auto")
	}
	if cfg.WorkDir != "" {
		args = append(args, "--cd", cfg.WorkDir)
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	args = append(args, cfg.ExtraBinaryFlags...)
	args = append(args, "--", prompt)
	return args
}

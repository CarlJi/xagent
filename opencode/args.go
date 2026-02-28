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
	"encoding/json"

	"github.com/goplus/xagent"
)

func buildRunArgs(cfg xagent.SessionConfig, prompt, sessionID string, fork bool) []string {
	args := []string{"run", "--format", "json"}
	if cfg.WorkDir != "" {
		args = append(args, "--dir", cfg.WorkDir)
	}
	if sessionID != "" {
		args = append(args, "--session", sessionID)
	}
	if fork {
		args = append(args, "--fork-session")
	}
	args = append(args, cfg.ExtraBinaryFlags...)
	args = append(args, "--", prompt)
	return args
}

func buildConfigContent(cfg xagent.SessionConfig) string {
	p := map[string]any{"*": "allow", "doom_loop": "deny"}
	if cfg.Permission == xagent.PermReadOnly {
		p = map[string]any{"*": "deny", "read": "allow", "doom_loop": "deny"}
	}
	b, _ := json.Marshal(map[string]any{"permission": p})
	return string(b)
}

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
	"strings"
	"time"

	"github.com/goplus/xagent"
)

func str(v any) string { s, _ := v.(string); return s }
func intV(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	default:
		return 0
	}
}
func boolV(v any) bool { b, _ := v.(bool); return b }

func parseSSELine(line []byte) []xagent.Event {
	text := strings.TrimSpace(string(line))
	if !strings.HasPrefix(text, "data:") {
		return nil
	}
	payload := strings.TrimSpace(strings.TrimPrefix(text, "data:"))
	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return nil
	}
	typ, _ := m["type"].(string)
	now := time.Now()
	switch typ {
	case "step_start":
		sid, _ := m["sessionID"].(string)
		return []xagent.Event{xagent.InitEvent{SessionID: sid, AgentName: "opencode", Timestamp: now}}
	case "text_delta":
		return []xagent.Event{xagent.TextEvent{Delta: str(m["delta"]), Timestamp: now}}
	case "tool_use":
		in, err := json.Marshal(m["input"])
		if err != nil {
			return []xagent.Event{xagent.ErrorEvent{
				Message:   "failed to marshal tool input: " + err.Error(),
				Code:      xagent.ErrExecution,
				Fatal:     false,
				Timestamp: now,
			}}
		}
		return []xagent.Event{xagent.ToolStartEvent{ToolName: str(m["name"]), CallID: str(m["id"]), Input: in, Timestamp: now}}
	case "tool_result":
		return []xagent.Event{xagent.ToolEndEvent{ToolName: str(m["name"]), CallID: str(m["id"]), Output: str(m["output"]), IsError: boolV(m["is_error"]), Timestamp: now}}
	case "step_finish":
		tokens, _ := m["tokens"].(map[string]any)
		return []xagent.Event{xagent.TurnCompleteEvent{InputTokens: intV(tokens["input"]), OutputTokens: intV(tokens["output"]), StopReason: str(m["stop_reason"]), Timestamp: now}}
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return []xagent.Event{xagent.ErrorEvent{
			Message:   "failed to marshal raw event: " + err.Error(),
			Code:      xagent.ErrExecution,
			Fatal:     false,
			Timestamp: now,
		}}
	}
	return []xagent.Event{xagent.RawEvent{AgentName: "opencode", RawJSON: raw, Timestamp: now}}
}

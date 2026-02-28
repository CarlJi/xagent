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

package gemini

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/internal/ndjson"
)

func parseNDJSON(data []byte) ([]xagent.Event, string, error) {
	s := ndjson.NewScanner(bytes.NewReader(data))
	var out []xagent.Event
	var id string
	for s.Scan() {
		line := append([]byte(nil), s.Bytes()...)
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, "", err
		}
		typ, _ := m["type"].(string)
		now := time.Now()
		switch typ {
		case "init":
			id, _ = m["session_id"].(string)
			out = append(out, xagent.InitEvent{SessionID: id, Model: str(m["model"]), AgentName: "gemini", Timestamp: now})
		case "content", "text":
			out = append(out, xagent.TextEvent{Delta: str(m["delta"]), Timestamp: now})
		case "tool_use":
			in, err := json.Marshal(m["input"])
			if err != nil {
				out = append(out, xagent.ErrorEvent{
					Message:   "failed to marshal tool input: " + err.Error(),
					Code:      xagent.ErrExecution,
					Fatal:     false,
					Timestamp: now,
				})
			} else {
				out = append(out, xagent.ToolStartEvent{ToolName: str(m["name"]), CallID: str(m["id"]), Input: in, Timestamp: now})
			}
		case "tool_result":
			out = append(out, xagent.ToolEndEvent{ToolName: str(m["name"]), CallID: str(m["id"]), Output: str(m["output"]), IsError: boolV(m["is_error"]), Timestamp: now})
		case "result":
			stats, _ := m["stats"].(map[string]any)
			out = append(out, xagent.TurnCompleteEvent{InputTokens: intV(stats["input_tokens"]), OutputTokens: intV(stats["output_tokens"]), StopReason: str(m["stop_reason"]), Timestamp: now})
		default:
			out = append(out, xagent.RawEvent{AgentName: "gemini", RawJSON: line, Timestamp: now})
		}
	}
	return out, id, s.Err()
}

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

func parseNDJSONLine(line []byte) []xagent.Event {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return nil
	}
	typ, _ := m["type"].(string)
	now := time.Now()
	switch typ {
	case "init":
		id, _ := m["session_id"].(string)
		return []xagent.Event{xagent.InitEvent{SessionID: id, Model: str(m["model"]), AgentName: "gemini", Timestamp: now}}
	case "content", "text":
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
	case "result":
		stats, _ := m["stats"].(map[string]any)
		return []xagent.Event{xagent.TurnCompleteEvent{InputTokens: intV(stats["input_tokens"]), OutputTokens: intV(stats["output_tokens"]), StopReason: str(m["stop_reason"]), Timestamp: now}}
	}
	return []xagent.Event{xagent.RawEvent{AgentName: "gemini", RawJSON: append([]byte(nil), line...), Timestamp: now}}
}

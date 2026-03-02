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

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/internal/ndjson"
)

func parseJSONL(data []byte) ([]xagent.Event, string, error) {
	s := ndjson.NewScanner(bytes.NewReader(data))
	var events []xagent.Event
	var threadID string
	for s.Scan() {
		line := append([]byte(nil), s.Bytes()...)
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, "", err
		}
		typ, _ := m["type"].(string)
		now := time.Now()
		switch typ {
		case "thread.started":
			threadID, _ = m["thread_id"].(string)
			events = append(events, xagent.InitEvent{SessionID: threadID, AgentName: "codex", Timestamp: now})
		case "item.completed":
			item, _ := m["item"].(map[string]any)
			if txt, _ := item["text"].(string); txt != "" {
				events = append(events, xagent.TextEvent{Delta: txt, Timestamp: now})
			}
		case "turn.completed":
			usage, _ := m["usage"].(map[string]any)
			events = append(events, xagent.TurnCompleteEvent{InputTokens: intNum(usage["input_tokens"]), OutputTokens: intNum(usage["output_tokens"]), StopReason: str(m["stop_reason"]), Timestamp: now})
		case "turn.failed":
			events = append(events, xagent.ErrorEvent{Message: str(m["message"]), Code: xagent.ErrExecution, Fatal: false, Timestamp: now})
		case "error":
			events = append(events, xagent.ErrorEvent{Message: str(m["message"]), Code: xagent.ErrExecution, Fatal: true, Timestamp: now})
		default:
			events = append(events, xagent.RawEvent{AgentName: "codex", RawJSON: line, Timestamp: now})
		}
	}
	return events, threadID, s.Err()
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func intNum(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func parseJSONLLine(line []byte) []xagent.Event {
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
	case "thread.started":
		threadID, _ := m["thread_id"].(string)
		return []xagent.Event{xagent.InitEvent{SessionID: threadID, AgentName: "codex", Timestamp: now}}
	case "item.completed":
		item, _ := m["item"].(map[string]any)
		if txt, _ := item["text"].(string); txt != "" {
			return []xagent.Event{xagent.TextEvent{Delta: txt, Timestamp: now}}
		}
	case "turn.completed":
		usage, _ := m["usage"].(map[string]any)
		return []xagent.Event{xagent.TurnCompleteEvent{InputTokens: intNum(usage["input_tokens"]), OutputTokens: intNum(usage["output_tokens"]), StopReason: str(m["stop_reason"]), Timestamp: now}}
	case "turn.failed":
		return []xagent.Event{xagent.ErrorEvent{Message: str(m["message"]), Code: xagent.ErrExecution, Fatal: false, Timestamp: now}}
	case "error":
		return []xagent.Event{xagent.ErrorEvent{Message: str(m["message"]), Code: xagent.ErrExecution, Fatal: true, Timestamp: now}}
	}
	return []xagent.Event{xagent.RawEvent{AgentName: "codex", RawJSON: append([]byte(nil), line...), Timestamp: now}}
}

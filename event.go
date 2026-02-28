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

import "time"

// Event is the sealed interface for all stream events emitted by an agent.
// Use a type switch to extract the concrete event type.
type Event interface {
	// StreamEventKind returns the kind of this event for quick classification.
	StreamEventKind() EventKind
	// StreamTimestamp returns the wall-clock time at which the event was produced.
	StreamTimestamp() time.Time
}

// EventKind is a numeric discriminant identifying the type of a stream Event.
type EventKind int

const (
	EventKindUnknown      EventKind = 0  // unrecognised event
	EventKindInit         EventKind = 1  // session initialisation metadata
	EventKindText         EventKind = 2  // incremental text delta
	EventKindThinking     EventKind = 3  // incremental thinking / reasoning delta
	EventKindToolStart    EventKind = 4  // agent began a tool call
	EventKindToolEnd      EventKind = 5  // agent received a tool result
	EventKindTurnComplete EventKind = 6  // agent finished a full turn
	EventKindError        EventKind = 7  // error reported by the backend
	EventKindRaw          EventKind = 99 // raw unparsed JSON line from the backend
)

// InitEvent is emitted once at the start of a session and carries session-level metadata.
type InitEvent struct {
	SessionID  string   // backend-assigned session identifier
	Model      string   // LLM model used for this session
	AgentName  string   // agent backend name (e.g. "claude")
	ToolNames  []string // list of tool names available to the agent
	CLIVersion string   // version string of the underlying CLI binary
	Timestamp  time.Time
}

func (e InitEvent) StreamEventKind() EventKind { return EventKindInit }
func (e InitEvent) StreamTimestamp() time.Time { return e.Timestamp }

// TextEvent carries an incremental text chunk produced by the agent.
type TextEvent struct {
	Delta     string // partial text to append to the running response
	Timestamp time.Time
}

func (e TextEvent) StreamEventKind() EventKind { return EventKindText }
func (e TextEvent) StreamTimestamp() time.Time { return e.Timestamp }

// ThinkingEvent carries an incremental chunk of the agent's internal reasoning trace
// (only emitted by backends that support extended thinking).
type ThinkingEvent struct {
	Delta     string // partial thinking text to append
	Timestamp time.Time
}

func (e ThinkingEvent) StreamEventKind() EventKind { return EventKindThinking }
func (e ThinkingEvent) StreamTimestamp() time.Time { return e.Timestamp }

// ToolStartEvent is emitted when the agent begins executing a tool call.
type ToolStartEvent struct {
	ToolName  string // name of the tool being invoked
	CallID    string // unique identifier for this tool call, matched by ToolEndEvent
	Input     []byte // raw JSON-encoded input arguments supplied by the agent
	Source    string // source of the tool (e.g. "builtin", "mcp")
	MCPServer string // MCP server name when Source is "mcp"
	Timestamp time.Time
}

func (e ToolStartEvent) StreamEventKind() EventKind { return EventKindToolStart }
func (e ToolStartEvent) StreamTimestamp() time.Time { return e.Timestamp }

// ToolEndEvent is emitted when the agent receives the result of a tool call.
type ToolEndEvent struct {
	ToolName  string // name of the tool that was invoked
	CallID    string // identifier matching the corresponding ToolStartEvent
	Output    string // tool result as a string
	IsError   bool   // true if the tool call returned an error result
	Timestamp time.Time
}

func (e ToolEndEvent) StreamEventKind() EventKind { return EventKindToolEnd }
func (e ToolEndEvent) StreamTimestamp() time.Time { return e.Timestamp }

// TurnCompleteEvent is emitted when the agent finishes a single turn and provides
// token usage, cost, and stop reason information.
type TurnCompleteEvent struct {
	InputTokens  int           // total input tokens consumed in this turn
	OutputTokens int           // total output tokens produced in this turn
	CostUSD      float64       // estimated cost in USD for this turn
	Duration     time.Duration // wall-clock duration of the turn
	StopReason   string        // reason the turn ended (e.g. "end_turn", "max_tokens")
	Timestamp    time.Time
}

func (e TurnCompleteEvent) StreamEventKind() EventKind { return EventKindTurnComplete }
func (e TurnCompleteEvent) StreamTimestamp() time.Time { return e.Timestamp }

// ErrorEvent is emitted when the agent backend reports an error condition.
type ErrorEvent struct {
	Message   string    // human-readable error description
	Code      ErrorCode // machine-readable error code
	Fatal     bool      // if true, the stream is terminated after this event
	Timestamp time.Time
}

func (e ErrorEvent) StreamEventKind() EventKind { return EventKindError }
func (e ErrorEvent) StreamTimestamp() time.Time { return e.Timestamp }

// RawEvent carries an unparsed JSON line that did not match any known event type.
// It is useful for diagnostics or handling future backend output formats.
type RawEvent struct {
	AgentName string // name of the agent that emitted this line
	RawJSON   []byte // verbatim JSON bytes from the agent output stream
	Timestamp time.Time
}

func (e RawEvent) StreamEventKind() EventKind { return EventKindRaw }
func (e RawEvent) StreamTimestamp() time.Time { return e.Timestamp }

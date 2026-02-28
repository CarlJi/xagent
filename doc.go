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

// Package xagent provides a unified SDK for orchestrating AI coding agents.
//
// It defines core abstractions (Agent, Session, Stream, Event) that allow
// callers to start sessions, send prompts, and consume streaming events
// from multiple backend agents (Claude Code, Codex CLI, Gemini CLI, OpenCode)
// through a consistent interface.
//
// Backend implementations live in sub-packages (claude, codex, gemini, opencode).
// The xagenttest package provides mock executors and test helpers.
package xagent

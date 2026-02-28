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

import "fmt"

// ErrorCode is a machine-readable string that identifies the class of an AgentError.
type ErrorCode string

// Predefined ErrorCode values used throughout the SDK.
const (
	ErrUnknown          ErrorCode = "unknown"           // unclassified error
	ErrBinaryNotFound   ErrorCode = "binary_not_found"  // agent CLI binary not found or not executable
	ErrAuth             ErrorCode = "auth"              // missing or invalid API credentials
	ErrInvalidConfig    ErrorCode = "invalid_config"    // invalid SessionConfig or Option value
	ErrExecution        ErrorCode = "execution"         // subprocess execution or I/O failure
	ErrParse            ErrorCode = "parse"             // failed to parse agent output
	ErrRateLimit        ErrorCode = "rate_limit"        // API rate limit reached
	ErrContextTooLong   ErrorCode = "context_too_long"  // prompt exceeds model context window
	ErrPermissionDenied ErrorCode = "permission_denied" // agent lacks permission to perform action
	ErrVersionIncompat  ErrorCode = "version_incompat"  // CLI version is not compatible with the SDK
	ErrProcessCrash     ErrorCode = "process_crash"     // agent subprocess terminated unexpectedly
	ErrTimeout          ErrorCode = "timeout"           // operation timed out
	ErrMaxTurns         ErrorCode = "max_turns"         // MaxTurns limit reached
	ErrMaxBudget        ErrorCode = "max_budget"        // cost or token budget exhausted
)

// AgentError is the structured error type returned by all SDK operations.
// It implements the error and Unwrap interfaces, compatible with errors.Is and errors.As.
type AgentError struct {
	Code    ErrorCode // machine-readable error classification
	Agent   string    // name of the agent backend that produced the error
	Message string    // human-readable description
	Cause   error     // underlying error, if any
}

func (e *AgentError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AgentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

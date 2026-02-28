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

package xagenttest

import (
	"context"

	"github.com/goplus/xagent"
)

// MockAgent is a test double for xagent.Agent that returns a fixed Session and error.
type MockAgent struct {
	AgentName string
	Caps      xagent.Capabilities
	Session   xagent.Session
	Err       error
}

func (m *MockAgent) Name() string { return m.AgentName }

func (m *MockAgent) Capabilities() xagent.Capabilities { return m.Caps }

func (m *MockAgent) Validate(ctx context.Context) error { return m.Err }

func (m *MockAgent) Start(ctx context.Context, cfg xagent.SessionConfig) (xagent.Session, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Session, nil
}

func (m *MockAgent) Close(ctx context.Context) error { return nil }

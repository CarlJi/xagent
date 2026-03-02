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
	"testing"

	"github.com/goplus/xagent"
)

// DrainStream consumes all events from s and returns them together with any error.
// It closes the stream before returning.
func DrainStream(ctx context.Context, s xagent.Stream) ([]xagent.Event, error) {
	defer s.Close()
	var events []xagent.Event
	for s.Next(ctx) {
		events = append(events, s.Event())
	}
	return events, s.Err()
}

// AssertEvents drains from the stream and verifies the expected event kinds in order.
func AssertEvents(t *testing.T, ctx context.Context, s xagent.Stream, expectedKinds ...xagent.EventKind) []xagent.Event {
	t.Helper()
	events, err := DrainStream(ctx, s)
	if err != nil {
		t.Fatalf("DrainStream error: %v", err)
	}
	if len(events) != len(expectedKinds) {
		t.Fatalf("expected %d events, got %d: %v", len(expectedKinds), len(events), eventKinds(events))
	}
	for i, ek := range expectedKinds {
		if got := events[i].StreamEventKind(); got != ek {
			t.Errorf("event[%d]: expected kind %d, got %d", i, ek, got)
		}
	}
	return events
}

func eventKinds(events []xagent.Event) []xagent.EventKind {
	out := make([]xagent.EventKind, len(events))
	for i, e := range events {
		out[i] = e.StreamEventKind()
	}
	return out
}

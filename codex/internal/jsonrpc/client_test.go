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

package jsonrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientRequestNotifyAndSubscribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if _, ok := req["id"]; ok {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": req["id"], "result": map[string]any{"thread_id": "th_1", "text": "ok"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	c := NewClient(NewHTTPTransport(srv.URL))
	defer c.Close()

	var out map[string]any
	if err := c.Request(context.Background(), "conversation.send", map[string]any{"prompt": "hi"}, &out); err != nil {
		t.Fatal(err)
	}
	if out["thread_id"] != "th_1" {
		t.Fatalf("unexpected response: %#v", out)
	}
	if err := c.Notify(context.Background(), "noop", map[string]any{"a": 1}); err != nil {
		t.Fatal(err)
	}

	ch := c.Subscribe()
	c.PublishNotification(json.RawMessage(`{"type":"notice"}`))
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("did not receive notification")
	}
	c.Unsubscribe(ch)
}

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

package session

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goplus/xagent"
)

type fakeSession struct{}

func (f *fakeSession) ID() string { return "id" }
func (f *fakeSession) Send(ctx context.Context, prompt string) (xagent.Stream, error) {
	return xagent.NewSliceStream(nil, nil), nil
}
func (f *fakeSession) IsHealthy(ctx context.Context) bool { return true }
func (f *fakeSession) Close(ctx context.Context) error    { return nil }

func TestManagerConcurrentGetSingleFactory(t *testing.T) {
	m := NewManager()
	key := NewSessionKey("user", "repo", "pr", "1")
	var n atomic.Int32

	factory := func() (xagent.Session, error) {
		n.Add(1)
		time.Sleep(10 * time.Millisecond)
		return &fakeSession{}, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := m.Get(key, factory)
			if err != nil {
				t.Errorf("get err: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := n.Load(); got != 1 {
		t.Fatalf("factory called %d times", got)
	}
}

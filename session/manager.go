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
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/goplus/xagent"
)

// SessionKey is an opaque composite key used to identify a cached session
// within a Manager. Create one with NewSessionKey.
type SessionKey string

// NewSessionKey constructs a SessionKey by URL-encoding and joining parts with ":".
// Use multiple parts to represent hierarchical keys (e.g. orgID, repoID, userID).
func NewSessionKey(parts ...string) SessionKey {
	encoded := make([]string, 0, len(parts))
	for _, p := range parts {
		encoded = append(encoded, url.QueryEscape(strings.TrimSpace(p)))
	}
	return SessionKey(strings.Join(encoded, ":"))
}

type item struct {
	session      xagent.Session
	lastActiveAt time.Time
}

type pendingGet struct {
	done    chan struct{}
	session xagent.Session
	err     error
}

// Manager is a thread-safe cache of xagent.Session instances, keyed by SessionKey.
// It is designed for use in long-running server processes that serve many concurrent requests.
type Manager struct {
	mu      sync.Mutex
	items   map[SessionKey]*item
	pending map[SessionKey]*pendingGet
}

// NewManager creates an empty Manager ready for use.
func NewManager() *Manager {
	return &Manager{items: map[SessionKey]*item{}, pending: map[SessionKey]*pendingGet{}}
}

// Get returns the session associated with key, creating a new one via factory if none exists.
// Each successful call updates the session's last-active timestamp.
func (m *Manager) Get(key SessionKey, factory func() (xagent.Session, error)) (xagent.Session, error) {
	for {
		now := time.Now()
		m.mu.Lock()
		if it, ok := m.items[key]; ok {
			it.lastActiveAt = now
			sess := it.session
			m.mu.Unlock()
			return sess, nil
		}
		if p, ok := m.pending[key]; ok {
			done := p.done
			m.mu.Unlock()
			<-done
			continue
		}
		p := &pendingGet{done: make(chan struct{})}
		m.pending[key] = p
		m.mu.Unlock()

		sess, err := factory()

		m.mu.Lock()
		delete(m.pending, key)
		p.session = sess
		p.err = err
		if err == nil {
			m.items[key] = &item{session: sess, lastActiveAt: now}
		}
		close(p.done)
		m.mu.Unlock()

		if err != nil {
			return nil, err
		}
		return sess, nil
	}
}

// Delete removes the session with key from the cache and closes it.
// It is a no-op if the key does not exist.
func (m *Manager) Delete(ctx context.Context, key SessionKey) error {
	m.mu.Lock()
	it, ok := m.items[key]
	if ok {
		delete(m.items, key)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return it.session.Close(ctx)
}

// Evict closes and removes all sessions that have been idle for longer than olderThan
// or that report IsHealthy as false. It returns the number of sessions evicted.
func (m *Manager) Evict(ctx context.Context, olderThan time.Duration) int {
	threshold := time.Now().Add(-olderThan)
	type candidate struct {
		key     SessionKey
		session xagent.Session
		idle    bool
	}
	var candidates []candidate
	m.mu.Lock()
	for k, it := range m.items {
		candidates = append(candidates, candidate{
			key:     k,
			session: it.session,
			idle:    it.lastActiveAt.Before(threshold),
		})
	}
	m.mu.Unlock()

	keys := make([]SessionKey, 0, len(candidates))
	for _, c := range candidates {
		if c.idle || !c.session.IsHealthy(ctx) {
			keys = append(keys, c.key)
		}
	}

	var toClose []xagent.Session
	m.mu.Lock()
	for _, key := range keys {
		if it, ok := m.items[key]; ok {
			toClose = append(toClose, it.session)
			delete(m.items, key)
		}
	}
	m.mu.Unlock()

	for _, s := range toClose {
		_ = s.Close(ctx)
	}
	return len(toClose)
}

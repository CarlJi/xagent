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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Transport is the pluggable HTTP-level transport used by a Client to dispatch JSON-RPC requests.
type Transport interface {
	Do(ctx context.Context, payload []byte) ([]byte, error)
	Close() error
}

// HTTPTransport implements Transport over a standard HTTP POST endpoint.
type HTTPTransport struct {
	endpoint string
	hc       *http.Client
}

// NewHTTPTransport creates an HTTPTransport targeting the given endpoint URL.
func NewHTTPTransport(endpoint string) *HTTPTransport {
	return &HTTPTransport{endpoint: endpoint, hc: &http.Client{Timeout: 15 * time.Second}}
}

func (t *HTTPTransport) Do(ctx context.Context, payload []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("jsonrpc http %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (t *HTTPTransport) Close() error { return nil }

// Client is a lightweight JSON-RPC 2.0 client with retry logic and pub/sub support.
type Client struct {
	transport Transport
	nextID    atomic.Int64

	mu          sync.RWMutex
	subscribers map[chan json.RawMessage]struct{}
}

// NewClient creates a Client using the provided Transport.
// Pass nil to create a client without a transport; set one later with SetTransport.
func NewClient(transport Transport) *Client {
	return &Client{transport: transport, subscribers: map[chan json.RawMessage]struct{}{}}
}

// SetTransport replaces the client's transport. Safe to call concurrently.
func (c *Client) SetTransport(t Transport) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.transport = t
}

// Request sends a JSON-RPC request with the given method and params, then decodes
// the result into out. It retries up to 3 times with exponential back-off.
func (c *Client) Request(ctx context.Context, method string, params any, out any) error {
	id := c.nextID.Add(1)
	req := map[string]any{"id": id, "method": method, "params": params}
	p, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal jsonrpc request: %w", err)
	}

	body, err := c.doWithRetry(ctx, p)
	if err != nil {
		return err
	}
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if rawErr, ok := resp["error"]; ok && len(rawErr) > 0 {
		return fmt.Errorf("jsonrpc error: %s", string(rawErr))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp["result"], out)
}

// Notify sends a JSON-RPC notification (no response expected).
func (c *Client) Notify(ctx context.Context, method string, params any) error {
	req := map[string]any{"method": method, "params": params}
	p, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal jsonrpc notification: %w", err)
	}
	_, err = c.doWithRetry(ctx, p)
	return err
}

// Subscribe returns a channel that receives server-push notifications.
// Call Unsubscribe to stop receiving messages and close the channel.
func (c *Client) Subscribe() chan json.RawMessage {
	ch := make(chan json.RawMessage, 32)
	c.mu.Lock()
	c.subscribers[ch] = struct{}{}
	c.mu.Unlock()
	return ch
}

// Unsubscribe removes ch from the subscriber set and closes it.
func (c *Client) Unsubscribe(ch chan json.RawMessage) {
	c.mu.Lock()
	if _, ok := c.subscribers[ch]; ok {
		delete(c.subscribers, ch)
		close(ch)
	}
	c.mu.Unlock()
}

// PublishNotification delivers msg to all current subscribers.
func (c *Client) PublishNotification(msg json.RawMessage) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for ch := range c.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (c *Client) doWithRetry(ctx context.Context, payload []byte) ([]byte, error) {
	for i := 0; i < 3; i++ {
		c.mu.RLock()
		transport := c.transport
		c.mu.RUnlock()
		if transport == nil {
			return nil, fmt.Errorf("jsonrpc transport is nil")
		}
		body, err := transport.Do(ctx, payload)
		if err == nil {
			return body, nil
		}
		backoff := time.Duration(math.Min(2000, 100*math.Pow(2, float64(i)))) * time.Millisecond
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, fmt.Errorf("jsonrpc request failed after retries")
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for ch := range c.subscribers {
		close(ch)
		delete(c.subscribers, ch)
	}
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}

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

package ndjson

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestScannerBasic(t *testing.T) {
	r := strings.NewReader("\n {\"a\":1}\n\n{\"b\":2}\n")
	s := NewScanner(r)

	var lines []string
	for s.Scan() {
		lines = append(lines, string(s.Bytes()))
	}
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestScannerMaxLine(t *testing.T) {
	line := strings.Repeat("a", MaxLineSize-1)
	r := strings.NewReader(line + "\n")
	s := NewScanner(r)
	if !s.Scan() {
		t.Fatalf("expected one line")
	}
	if got := len(s.Bytes()); got != len(line) {
		t.Fatalf("unexpected line size: %d", got)
	}
}

func TestScannerOverLimit(t *testing.T) {
	line := strings.Repeat("a", MaxLineSize+1)
	r := strings.NewReader(line + "\n")
	s := NewScanner(r)
	if s.Scan() {
		t.Fatalf("expected scan to fail")
	}
	if s.Err() == nil {
		t.Fatalf("expected error for over-limit line")
	}
}

type brokenReader struct{}

func (brokenReader) Read(_ []byte) (int, error) {
	return 0, errors.New("boom")
}

func TestScannerReaderError(t *testing.T) {
	s := NewScanner(brokenReader{})
	if s.Scan() {
		t.Fatalf("expected false")
	}
	if s.Err() == nil {
		t.Fatalf("expected error")
	}
}

func TestScannerUTF8Boundary(t *testing.T) {
	data := bytes.NewBuffer(nil)
	data.WriteString("你好")
	data.WriteByte('\n')
	s := NewScanner(data)
	if !s.Scan() {
		t.Fatalf("expected line")
	}
	if _, err := io.Copy(io.Discard, bytes.NewReader(s.Bytes())); err != nil {
		t.Fatalf("copy failed: %v", err)
	}
}

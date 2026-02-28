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
	"bufio"
	"bytes"
	"io"
)

// MaxLineSize is the maximum byte size of a single NDJSON line the Scanner will buffer.
const MaxLineSize = 10 << 20

// Scanner wraps bufio.Scanner to read newline-delimited JSON lines from an io.Reader,
// automatically skipping blank lines and enlarging the internal buffer up to MaxLineSize.
type Scanner struct {
	scanner *bufio.Scanner
	bytes   []byte
}

// NewScanner creates a Scanner that reads NDJSON from r.
func NewScanner(r io.Reader) *Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64<<10), MaxLineSize)
	return &Scanner{scanner: s}
}

// Scan advances the scanner to the next non-blank JSON line.
// It returns false when there are no more lines or an error occurred.
func (s *Scanner) Scan() bool {
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		s.bytes = append(s.bytes[:0], line...)
		return true
	}
	return false
}

// Bytes returns the current line bytes (valid until the next call to Scan).
func (s *Scanner) Bytes() []byte {
	return s.bytes
}

// Err returns the first error encountered while scanning, or nil on clean EOF.
func (s *Scanner) Err() error {
	return s.scanner.Err()
}

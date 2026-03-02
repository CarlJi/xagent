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
	"strings"
	"testing"
)

func BenchmarkScannerThroughput(b *testing.B) {
	line := `{"type":"text","delta":"hello"}` + "\n"
	data := bytes.Repeat([]byte(line), 2_500_000)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewScanner(bytes.NewReader(data))
		for s.Scan() {
		}
		if err := s.Err(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScannerLargeLine(b *testing.B) {
	line := strings.Repeat("x", MaxLineSize-10) + "\n"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewScanner(strings.NewReader(line))
		if !s.Scan() {
			b.Fatal("scan failed")
		}
	}
}

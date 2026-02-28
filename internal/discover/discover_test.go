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

package discover

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindWithExplicitPath(t *testing.T) {
	d := t.TempDir()
	name := "mybin"
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	f := filepath.Join(d, name)
	content := []byte("#!/bin/sh\n")
	if runtime.GOOS == "windows" {
		content = []byte("@echo off\r\n")
	}
	if err := os.WriteFile(f, content, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := FindWithPath("mybin", f)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p == "" {
		t.Fatalf("expected path")
	}
}

func TestFindMissing(t *testing.T) {
	_, err := Find("definitely-not-exists-xagent")
	if err == nil {
		t.Fatalf("expected err")
	}
}

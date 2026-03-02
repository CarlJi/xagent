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

package docker

import (
	"context"
	"testing"
)

func TestPathRemap(t *testing.T) {
	e, err := New(WithImage("test:latest"), WithContainerName("test-container"), WithPathRemap(map[string]string{"/host/project": "/workspace"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := e.mapWorkDir("/host/project/sub"); got != "/workspace/sub" {
		t.Fatalf("unexpected remap: %s", got)
	}
	if e.IsHealthy(context.Background()) {
		// no docker daemon in unit tests, just ensure method callable
	}
}

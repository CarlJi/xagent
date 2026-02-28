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

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/claude"
	"github.com/goplus/xagent/codex"
	"github.com/goplus/xagent/gemini"
	"github.com/goplus/xagent/opencode"
)

func main() {
	ctx := context.Background()
	agents := []xagent.Agent{
		claude.New(claude.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY"))),
		codex.New(codex.WithAPIKey(os.Getenv("OPENAI_API_KEY"))),
		gemini.New(gemini.WithAPIKey(os.Getenv("GEMINI_API_KEY"))),
		opencode.New(),
	}
	for _, a := range agents {
		fmt.Printf("- %s %+v\n", a.Name(), a.Capabilities())
	}
	_ = ctx
}

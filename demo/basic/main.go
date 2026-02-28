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
	"log"
	"os"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/claude"
)

func main() {
	ctx := context.Background()
	agent := claude.New(claude.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))
	defer agent.Close(ctx)

	if err := agent.Validate(ctx); err != nil {
		log.Fatal(err)
	}

	res, err := xagent.Run(ctx, agent, xagent.SessionConfig{
		Permission: xagent.PermReadOnly,
	}, "用一句话介绍 xagent")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Text)
}

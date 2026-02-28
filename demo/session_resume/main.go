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
	var agent xagent.Agent = claude.New(claude.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))
	defer agent.Close(ctx)

	sess, err := agent.Start(ctx, xagent.SessionConfig{Permission: xagent.PermAutoApprove})
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close(ctx)

	if _, err := xagent.CollectText(ctx, must(sess.Send(ctx, "你好"))); err != nil {
		log.Fatal(err)
	}
	sessionID := sess.ID()
	sess.Close(ctx)

	// Simulate a process restart: resume the session by its ID.
	resumed, err := agent.Start(ctx, xagent.SessionConfig{
		SessionID:  sessionID,
		Permission: xagent.PermAutoApprove,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer resumed.Close(ctx)

	text, err := xagent.CollectText(ctx, must(resumed.Send(ctx, "继续上次会话，回复一句话")))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(text)
}

func must(s xagent.Stream, err error) xagent.Stream {
	if err != nil {
		log.Fatal(err)
	}
	return s
}

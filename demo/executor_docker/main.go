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

// _examples/docker_xgopilot/main.go
//
// Validates the xagent SDK using the goplusorg/codeagent:v0.9.6.1 image.
// Simulates the real xgopilot production flow: long-lived Docker container with claude CLI driven by docker exec.
//
// Prerequisites:
//   1. docker pull goplusorg/codeagent:v0.9.6.1  (already available locally)
//   2. Set the ANTHROPIC_API_KEY environment variable
//   3. Provide a Go project directory to analyse
//
// Run:
//   go run ./ --project /path/to/your/go/project

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/goplus/xagent"
	"github.com/goplus/xagent/claude"
	"github.com/goplus/xagent/demo/executor_docker/docker"
)

func main() {
	projectDir := flag.String("project", ".", "待分析的项目目录（主机路径）")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// ══════════════════════════════════════════════════════════
	// 1. Create DockerExecutor — aligned with xgopilot production config
	// ══════════════════════════════════════════════════════════
	//
	// Actual xgopilot behaviour:
	//   docker run --init --rm -d --name claude-goplus-xagent-pr-1
	//     -v /path/to/project:/workspace -w /workspace
	//     --user codeagent
	//     goplusorg/codeagent:v0.9.6.1
	//     sleep infinity
	//
	// Each subsequent prompt is dispatched via docker exec to the claude CLI.
	// The container stays alive for the entire session lifetime.

	exec, err := docker.New(
		docker.WithImage("goplusorg/codeagent:v0.9.6.1"),
		docker.WithContainerName("xagent-sdk-test"), // deterministic name; reusable across processes
		docker.WithUser("codeagent"),                // non-root user pre-configured in the image
		docker.WithWorkDir("/workspace"),            // working directory inside the container
		docker.WithMounts([]docker.Mount{
			{Host: *projectDir, Container: "/workspace"}, // project directory mount
		}),
		docker.WithPathRemap(map[string]string{
			*projectDir: "/workspace", // host path → container path auto-remapping
		}),
		docker.WithInit(true),       // --init: ensures zombie processes are reaped
		docker.WithAutoRemove(true), // --rm: container is deleted automatically on Close()
		docker.WithLogger(logger),
	)
	if err != nil {
		log.Fatalf("failed to create DockerExecutor: %v", err)
	}
	defer exec.Close(ctx)

	// ══════════════════════════════════════════════════════════
	// 2. Create Claude Agent, injecting the DockerExecutor
	// ══════════════════════════════════════════════════════════
	//
	// Actual xgopilot behaviour:
	//   docker exec --user codeagent
	//     -e ANTHROPIC_API_KEY=<key>
	//     -e CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
	//     xagent-sdk-test
	//     claude --output-format stream-json --verbose
	//     --dangerously-skip-permissions -p <message>

	var agent xagent.Agent = claude.New(
		claude.WithExecutor(exec),
		claude.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
		claude.WithBaseURL(os.Getenv("ANTHROPIC_BASE_URL")),
		claude.WithLogger(logger),
		// Inject additional environment variables used by xgopilot in production
		claude.WithEnv(map[string]string{
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		}),
	)
	defer agent.Close(ctx)

	// Pre-flight check: verify the claude binary is available inside the container and its version is compatible.
	if err := agent.Validate(ctx); err != nil {
		log.Fatalf("pre-flight check failed: %v", err)
	}
	fmt.Println("✅ Pre-flight OK: claude binary available")
	fmt.Printf("   backend: %s, capabilities: %+v\n", agent.Name(), agent.Capabilities())

	// ══════════════════════════════════════════════════════════
	// Scenario A: single-turn call — equivalent to xgopilot "Issue Analysis"
	// ══════════════════════════════════════════════════════════
	//
	// When workspace.FreshSession=true xgopilot does not pass -c,
	// which is equivalent to a one-shot xagent.Run() call.

	fmt.Println("\n═══ 场景 A: 单轮调用（Issue 分析）═══")
	result, err := xagent.Run(ctx, agent, xagent.SessionConfig{
		WorkDir:    *projectDir,
		Permission: xagent.PermReadOnly, // read-only analysis; no modifications
		MaxTurns:   3,
	}, "列出这个项目有哪些 Go 源文件，简要描述项目结构")
	if err != nil {
		log.Fatalf("single-turn call failed: %v", err)
	}
	fmt.Println(result.Text)
	fmt.Printf("📊 %d in / %d out tokens, $%.4f, %v\n",
		result.Usage.InputTokens, result.Usage.OutputTokens,
		result.Usage.CostUSD, result.Duration)

	// ══════════════════════════════════════════════════════════
	// Scenario B: multi-turn + streaming — equivalent to xgopilot "PR Review"
	// ══════════════════════════════════════════════════════════
	//
	// Actual xgopilot behaviour:
	//   Turn 1: docker exec ... claude ... -p "review this PR"
	//   Turn 2: docker exec ... claude ... --resume <id> -p "focus on error handling"
	//
	// Key difference: in Docker mode each docker exec is an independent subprocess;
	// a persistent stdin connection cannot be maintained. The SDK detects DockerExecutor
	// and automatically switches to --resume mode for subsequent turns.

	fmt.Println("\n═══ 场景 B: 多轮 PR Review（流式）═══")
	sess, err := agent.Start(ctx, xagent.SessionConfig{
		WorkDir:      *projectDir,
		Permission:   xagent.PermAutoApprove, // write permission required for PR-fix scenarios
		SystemPrompt: "你是代码审查专家。用中文回答。分析要简洁精准。",
	})
	if err != nil {
		log.Fatalf("failed to start session: %v", err)
	}
	defer sess.Close(ctx)

	// --- Turn 1: stream consumption ---
	stream, err := sess.Send(ctx, "检查项目中是否有未处理的 error，列出前3个最严重的")
	if err != nil {
		log.Fatalf("Send failed: %v", err)
	}

	fmt.Println("--- Turn 1 (streaming) ---")
	var toolCount int
	for stream.Next(ctx) {
		switch e := stream.Event().(type) {
		case xagent.InitEvent:
			fmt.Printf("📎 session ID: %s (CLI %s)\n", e.SessionID, e.CLIVersion)
		case xagent.TextEvent:
			fmt.Print(e.Delta)
		case xagent.ToolStartEvent:
			toolCount++
			fmt.Printf("\n  🔧 %s(%s)\n", e.ToolName, truncate(string(e.Input), 60))
		case xagent.ToolEndEvent:
			if e.IsError {
				fmt.Printf("  ❌ %s failed: %s\n", e.ToolName, truncate(e.Output, 80))
			}
		case xagent.TurnCompleteEvent:
			fmt.Printf("\n📊 Turn done: %d in/%d out, $%.4f, reason=%s\n",
				e.InputTokens, e.OutputTokens, e.CostUSD, e.StopReason)
		case xagent.ErrorEvent:
			if e.Fatal {
				log.Fatalf("💀 fatal error: [%s] %s", e.Code, e.Message)
			}
			logger.Warn("⚠️ non-fatal error", "code", e.Code, "msg", e.Message)
		}
	}
	if err := stream.Err(); err != nil {
		log.Fatalf("stream read error: %v", err)
	}
	fmt.Printf("(total tool calls: %d)\n", toolCount)

	// --- Turn 2: follow-up question (same Session; SDK uses --resume automatically) ---
	fmt.Println("\n--- Turn 2 (follow-up) ---")
	result2, err := xagent.CollectResult(ctx,
		mustStream(sess.Send(ctx, "针对第一个问题，给出修复代码")))
	if err != nil {
		log.Fatalf("turn 2 failed: %v", err)
	}
	fmt.Println(result2.Text)
	fmt.Printf("📊 %d in / %d out, tool calls: %d\n",
		result2.Usage.InputTokens, result2.Usage.OutputTokens, len(result2.ToolCalls))

	// ══════════════════════════════════════════════════════════
	// Scenario C: session resume — equivalent to xgopilot "container reuse"
	// ══════════════════════════════════════════════════════════
	//
	// Actual xgopilot behaviour:
	//   After a process restart, a container with the same name is found still running
	//   (isContainerRunning) and reused directly.
	//   xagent achieves the same result via Session.ID() + Start(SessionConfig{SessionID: id}).

	fmt.Println("\n═══ Scenario C: session resume ═══")
	sessionID := sess.ID()
	fmt.Printf("📸 session ID: %s\n", sessionID)

	// Simulate a process restart: close the current session, then resume by ID.
	sess.Close(ctx)

	if agent.Capabilities().SessionResume {
		fmt.Println("🔄 resuming session...")
		resumed, err := agent.Start(ctx, xagent.SessionConfig{
			SessionID:  sessionID,
			WorkDir:    *projectDir,
			Permission: xagent.PermAutoApprove,
		})
		if err != nil {
			log.Fatalf("Resume failed: %v", err)
		}
		defer resumed.Close(ctx)

		text, err := xagent.CollectText(ctx,
			mustStream(resumed.Send(ctx, "总结一下我们之前讨论的所有问题")))
		if err != nil {
			log.Fatalf("post-resume call failed: %v", err)
		}
		fmt.Printf("✅ reply after resume:\n%s\n", text)
	} else {
		fmt.Println("⚠️ current backend does not support Resume")
	}

	// ══════════════════════════════════════════════════════════
	// Scenario D: health check — equivalent to xgopilot IsHealthy + GetSession
	// ══════════════════════════════════════════════════════════

	fmt.Println("\n═══ Scenario D: health check ═══")
	fmt.Printf("🐳 container healthy: %v\n", exec.IsHealthy(ctx))

	fmt.Println("\n🎉 all scenarios validated")
}

// ── helper functions ──

func mustStream(s xagent.Stream, err error) xagent.Stream {
	if err != nil {
		log.Fatalf("Send failed: %v", err)
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

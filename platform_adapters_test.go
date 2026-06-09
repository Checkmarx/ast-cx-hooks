package agenthooks_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
)

// TestPlatformAdaptersEndToEnd drives Droid, Gemini, and Windsurf adapters through
// Dispatch — these were previously tested only at the builder level, not through
// their seam. Blocking (exit-2) deny paths are covered separately by the subprocess
// test below, since ProcessE calls os.Exit(2). Here we cover non-blocking routes and
// the allow path of blocking routes (both return JSON without exiting).
func TestPlatformAdaptersEndToEnd(t *testing.T) {
	cases := []struct {
		name       string
		route      string
		register   func()
		stdin      string
		wantStdout []string
	}{
		// --- Droid ---
		{
			name:  "droid-stop interrupt",
			route: "droid-stop",
			register: func() {
				agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.Agent != agenthooks.AgentDroid {
						t.Fatalf("agent: %q", e.Agent)
					}
					if e.IsLooping() {
						return agenthooks.Resume()
					}
					return agenthooks.Interrupt("keep going")
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","hook_event_name":"Stop","stop_hook_active":false}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"keep going"`},
		},
		{
			name:  "droid-stop loop break",
			route: "droid-stop",
			register: func() {
				agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.IsLooping() {
						return agenthooks.Resume()
					}
					return agenthooks.Interrupt("should not fire")
				})
			},
			stdin:      `{"session_id":"s","hook_event_name":"Stop","stop_hook_active":true}`,
			wantStdout: []string{`"continue":true`},
		},
		{
			name:  "droid-subagent-stop interrupt",
			route: "droid-subagent-stop",
			register: func() {
				agenthooks.WhenSubagentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					return agenthooks.Interrupt("finish subtask")
				})
			},
			stdin:      `{"session_id":"s","hook_event_name":"SubagentStop","stop_hook_active":false}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"finish subtask"`},
		},
		{
			name:  "droid-pre-tool-use allow",
			route: "droid-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					if !e.IsShell() || e.Command != "ls" {
						t.Fatalf("expected shell ls, got kind=%q cmd=%q", e.Kind, e.Command)
					}
					return agenthooks.Allow()
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","tool_name":"Execute","tool_input":{"command":"ls"}}`,
			wantStdout: []string{`"permissionDecision":"allow"`},
		},
		{
			name:  "droid-pre-tool-use ask emits ask decision (not hard block)",
			route: "droid-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					return agenthooks.AskUser("confirm?")
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","tool_name":"Bash","tool_input":{"command":"rm"}}`,
			wantStdout: []string{`"permissionDecision":"ask"`, `"confirm?"`},
		},
		{
			name:  "gemini-before-tool shell command is extracted for gating",
			route: "gemini-before-tool",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					if !e.IsShell() || e.Command != "rm -rf /" {
						t.Fatalf("expected shell command 'rm -rf /', got kind=%q cmd=%q", e.Kind, e.Command)
					}
					return agenthooks.Deny("blocked")
				})
			},
			stdin:      `{"session_id":"s","tool_name":"run_shell_command","tool_input":{"command":"rm -rf /"}}`,
			wantStdout: []string{`"decision":"deny"`},
		},
		{
			name:  "droid-after-file-write annotate",
			route: "droid-after-file-write",
			register: func() {
				agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
					if e.FilePath != "/r/main.go" {
						t.Fatalf("file path: %q", e.FilePath)
					}
					return agenthooks.AnnotateWrite("run gofmt")
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","tool_name":"Create","tool_input":{"file_path":"/r/main.go","content":"package main"}}`,
			wantStdout: []string{`"additionalContext":"run gofmt"`},
		},
		{
			name:  "droid-user-prompt-submit reject",
			route: "droid-user-prompt-submit",
			register: func() {
				agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
					return agenthooks.RejectPrompt("blocked")
				})
			},
			stdin:      `{"session_id":"s","hook_event_name":"UserPromptSubmit","prompt":"hi"}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"blocked"`},
		},

		// --- Gemini ---
		{
			name:  "gemini-after-agent interrupt uses deny",
			route: "gemini-after-agent",
			register: func() {
				agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.Agent != agenthooks.AgentGemini {
						t.Fatalf("agent: %q", e.Agent)
					}
					return agenthooks.Interrupt("try again")
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","hook_event_name":"AfterAgent","prompt":"p","prompt_response":"r","stop_hook_active":false}`,
			wantStdout: []string{`"decision":"deny"`, `"reason":"try again"`},
		},
		{
			name:  "gemini-before-tool deny",
			route: "gemini-before-tool",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					return agenthooks.Deny("nope")
				})
			},
			stdin:      `{"session_id":"s","tool_name":"run_shell_command","tool_input":{"command":"rm"}}`,
			wantStdout: []string{`"decision":"deny"`, `"reason":"nope"`},
		},
		{
			name:  "gemini-after-file-tool annotate",
			route: "gemini-after-file-tool",
			register: func() {
				agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
					if e.FilePath != "/r/x.go" {
						t.Fatalf("file path: %q", e.FilePath)
					}
					return agenthooks.AnnotateWrite("note")
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","tool_name":"write_file","tool_input":{"file_path":"/r/x.go"},"tool_response":{}}`,
			wantStdout: []string{`"additionalContext":"note"`},
		},
		{
			name:  "gemini-before-agent enrich",
			route: "gemini-before-agent",
			register: func() {
				agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
					return agenthooks.EnrichPrompt("extra")
				})
			},
			stdin:      `{"session_id":"s","hook_event_name":"BeforeAgent","prompt":"hi"}`,
			wantStdout: []string{`"additionalContext":"extra"`},
		},

		// --- Cursor subagent loop detection (IsLooping must see loop_count) ---
		{
			name:  "cursor-subagent-stop looping is detected",
			route: "cursor-subagent-stop",
			register: func() {
				agenthooks.WhenSubagentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if !e.IsLooping() {
						t.Fatalf("expected IsLooping()=true at loop_count 5 (got AutoRetryCount=%d)", e.AutoRetryCount)
					}
					return agenthooks.Resume()
				})
			},
			stdin:      `{"conversation_id":"c","status":"completed","loop_count":5}`,
			wantStdout: []string{},
		},
		{
			name:  "cursor-subagent-stop followup when not looping",
			route: "cursor-subagent-stop",
			register: func() {
				agenthooks.WhenSubagentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.IsLooping() {
						return agenthooks.Resume()
					}
					return agenthooks.Interrupt("keep going")
				})
			},
			stdin:      `{"conversation_id":"c","status":"completed","loop_count":0}`,
			wantStdout: []string{`"followup_message":"keep going"`},
		},

		// --- Windsurf (non-blocking + allow paths) ---
		{
			name:  "windsurf-post-cascade-response acknowledges",
			route: "windsurf-post-cascade-response",
			register: func() {
				agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.Agent != agenthooks.AgentWindsurf {
						t.Fatalf("agent: %q", e.Agent)
					}
					return agenthooks.Interrupt("ignored — fire and forget")
				})
			},
			stdin:      `{"agent_action_name":"post_cascade_response","trajectory_id":"t","tool_info":{"response":"done"}}`,
			wantStdout: []string{},
		},
		{
			name:  "windsurf-pre-run-command allow",
			route: "windsurf-pre-run-command",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					if e.Command != "go test" {
						t.Fatalf("command: %q", e.Command)
					}
					return agenthooks.Allow()
				})
			},
			stdin:      `{"agent_action_name":"pre_run_command","trajectory_id":"t","tool_info":{"command_line":"go test","cwd":"/r"}}`,
			wantStdout: []string{},
		},
		{
			name:  "windsurf-pre-read-code allow",
			route: "windsurf-pre-read-code",
			register: func() {
				agenthooks.BeforeFileRead(func(e agenthooks.FileReadEvent) agenthooks.FileReadVerdict {
					return agenthooks.AllowRead()
				})
			},
			stdin:      `{"agent_action_name":"pre_read_code","trajectory_id":"t","tool_info":{"file_path":"/r/ok.go"}}`,
			wantStdout: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agenthooks.ClearRoutes()
			tc.register()

			stdoutBuf := pipeStdio(t, tc.stdin)

			origArgs := os.Args
			os.Args = []string{"hook", tc.route}
			defer func() { os.Args = origArgs }()

			agenthooks.Dispatch()

			out := stdoutBuf()
			var anyJSON map[string]any
			if err := json.Unmarshal([]byte(out), &anyJSON); err != nil {
				t.Fatalf("stdout is not valid JSON: %q (err=%v)", out, err)
			}
			for _, want := range tc.wantStdout {
				if !strings.Contains(out, want) {
					t.Fatalf("stdout missing %q\nfull output: %s", want, out)
				}
			}
		})
	}
}

// TestWindsurfBlockingExitsTwo covers the exit-2 blocking path that the in-process
// tests cannot: ProcessE/RunE calls os.Exit(2) on deny, which would kill the test
// process, so we re-exec this test as a child and assert the child exits 2 with the
// reason on stderr.
func TestWindsurfBlockingExitsTwo(t *testing.T) {
	if os.Getenv("AGENTHOOKS_TEST_CHILD") == "1" {
		agenthooks.ClearRoutes()
		agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
			return agenthooks.Deny("blocked by test")
		})
		os.Args = []string{"hook", "windsurf-pre-run-command"}
		agenthooks.Dispatch() // RunE will os.Exit(2) on the deny
		return
	}

	exe, err := os.Executable() // robust against os.Args[0] being clobbered by other tests
	if err != nil {
		t.Fatalf("locating test binary: %v", err)
	}
	cmd := exec.Command(exe, "-test.run=TestWindsurfBlockingExitsTwo")
	cmd.Env = append(os.Environ(), "AGENTHOOKS_TEST_CHILD=1")
	cmd.Stdin = strings.NewReader(`{"agent_action_name":"pre_run_command","trajectory_id":"t","tool_info":{"command_line":"rm -rf /","cwd":"/"}}`)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected an exec.ExitError, got %v (stderr: %s)", err, stderr.String())
	}
	if ee.ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %d (stderr: %s)", ee.ExitCode(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "blocked by test") {
		t.Fatalf("stderr missing deny reason: %s", stderr.String())
	}
}

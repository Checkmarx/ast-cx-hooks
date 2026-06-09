package agenthooks_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks"
)

// TestCopilotCLIRoutesEndToEnd drives Copilot-CLI-shaped stdin payloads through
// Dispatch for each unified hook and verifies the unified handler runs and emits
// the FLAT output shape the Copilot CLI expects — no hookSpecificOutput wrapper,
// modifiedArgs (not updatedInput), top-level decision — and that lowercase CLI
// tool names (bash, create) classify correctly.
func TestCopilotCLIRoutesEndToEnd(t *testing.T) {
	cases := []struct {
		name       string
		route      string
		register   func()
		stdin      string
		wantStdout []string // substrings that must appear
		notStdout  []string // substrings that must NOT appear (proves FLAT shape)
	}{
		{
			name:  "pre-tool-use deny is flat",
			route: "copilot-cli-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					if e.Agent != agenthooks.AgentCopilotCLI {
						t.Fatalf("agent: got %q want copilot-cli", e.Agent)
					}
					if !e.IsShell() || !strings.Contains(e.Command, "rm -rf") {
						t.Fatalf("expected shell command rm -rf, got kind=%q cmd=%q", e.Kind, e.Command)
					}
					return agenthooks.Deny("blocked")
				})
			},
			stdin: `{
				"hook_event_name":"PreToolUse","session_id":"s-1","cwd":"/repo",
				"tool_name":"bash","tool_input":{"command":"rm -rf /"}
			}`,
			wantStdout: []string{`"permissionDecision":"deny"`, `"permissionDecisionReason":"blocked"`},
			notStdout:  []string{"hookSpecificOutput"},
		},
		{
			name:  "pre-tool-use rewrite uses modifiedArgs not updatedInput",
			route: "copilot-cli-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					return agenthooks.AllowWithInput(json.RawMessage(`{"command":"ls -la"}`))
				})
			},
			stdin: `{
				"hook_event_name":"PreToolUse","session_id":"s-2",
				"tool_name":"bash","tool_input":{"command":"ls"}
			}`,
			wantStdout: []string{`"permissionDecision":"allow"`, `"modifiedArgs"`, `"ls -la"`},
			notStdout:  []string{"updatedInput", "hookSpecificOutput"},
		},
		{
			name:  "stop interrupt is top-level decision",
			route: "copilot-cli-stop",
			register: func() {
				agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.Agent != agenthooks.AgentCopilotCLI {
						t.Fatalf("agent: got %q want copilot-cli", e.Agent)
					}
					return agenthooks.Interrupt("run tests")
				})
			},
			stdin: `{
				"hook_event_name":"Stop","session_id":"s-3",
				"transcript_path":"/tmp/t.jsonl","stop_reason":"end_turn"
			}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"run tests"`},
			notStdout:  []string{"hookSpecificOutput"},
		},
		{
			name:  "after-file-write create annotate",
			route: "copilot-cli-after-file-write",
			register: func() {
				agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
					if e.Agent != agenthooks.AgentCopilotCLI || e.FilePath != "/repo/main.go" {
						t.Fatalf("unexpected event: %+v", e)
					}
					return agenthooks.AnnotateWrite("run go vet")
				})
			},
			stdin: `{
				"hook_event_name":"PostToolUse","session_id":"s-4",
				"tool_name":"create","tool_input":{"file_path":"/repo/main.go","content":"package main"},
				"tool_result":{"result_type":"success","text_result_for_llm":"ok"}
			}`,
			wantStdout: []string{`"additionalContext":"run go vet"`},
			notStdout:  []string{"hookSpecificOutput"},
		},
		{
			name:  "after-file-write skips non-write tool",
			route: "copilot-cli-after-file-write",
			register: func() {
				agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
					t.Fatal("handler should not fire for view tool")
					return agenthooks.AcceptWrite()
				})
			},
			stdin: `{
				"hook_event_name":"PostToolUse","session_id":"s-5",
				"tool_name":"view","tool_input":{"file_path":"/repo/main.go"},
				"tool_result":{"result_type":"success","text_result_for_llm":"ok"}
			}`,
			wantStdout: []string{},
		},
		{
			name:  "post-tool-use-failure annotate",
			route: "copilot-cli-post-tool-use-failure",
			register: func() {
				agenthooks.AfterToolFailure(func(e agenthooks.ToolFailureEvent) agenthooks.ToolFailureVerdict {
					if e.Agent != agenthooks.AgentCopilotCLI || e.Error == "" {
						t.Fatalf("unexpected event: %+v", e)
					}
					return agenthooks.AnnotateFailure("retry with smaller input")
				})
			},
			stdin: `{
				"hook_event_name":"PostToolUseFailure","session_id":"s-6",
				"tool_name":"bash","tool_input":{"command":"x"},"error":"exit status 1"
			}`,
			wantStdout: []string{`"additionalContext":"retry with smaller input"`},
			notStdout:  []string{"decision", "hookSpecificOutput"},
		},
		{
			name:  "user-prompt-submit is observational (reject ignored)",
			route: "copilot-cli-user-prompt-submit",
			register: func() {
				agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
					if e.Agent != agenthooks.AgentCopilotCLI {
						t.Fatalf("agent: got %q want copilot-cli", e.Agent)
					}
					return agenthooks.RejectPrompt("no secrets") // CLI cannot block; ignored
				})
			},
			stdin: `{
				"hook_event_name":"UserPromptSubmit","session_id":"s-7","prompt":"leak API_KEY=abc"
			}`,
			wantStdout: []string{},
			// CLI does not process this event's output, so nothing actionable is emitted.
			notStdout: []string{"decision", "continue", "stopReason", "permissionDecision"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agenthooks.ClearRoutes()
			tc.register()

			stdoutBuf := pipeStdio(t, tc.stdin)

			origArgs := os.Args
			os.Args = []string{"copilotclihook", tc.route}
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
			for _, not := range tc.notStdout {
				if strings.Contains(out, not) {
					t.Fatalf("stdout unexpectedly contains %q (FLAT shape expected)\nfull output: %s", not, out)
				}
			}
		})
	}
}

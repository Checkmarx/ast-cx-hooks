package agenthooks_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Checkmarx/ast-cx-hooks"
)

// TestCodexRoutesEndToEnd drives Codex-CLI-shaped stdin payloads through
// Dispatch for each unified hook Codex backs and verifies the unified handler
// runs and emits the Claude-style nested output shape (hookSpecificOutput,
// updatedInput, permissionDecision) — see codex/doc.go for the caveat that
// this schema is modeled from the published doc, not a captured payload.
func TestCodexRoutesEndToEnd(t *testing.T) {
	cases := []struct {
		name       string
		route      string
		register   func()
		stdin      string
		wantStdout []string // substrings that must appear
		notStdout  []string // substrings that must NOT appear
		wantEmpty  bool     // stdout must be exactly empty (plain approve; see runPreToolUse)
	}{
		{
			name:  "codex-stop interrupt",
			route: "codex-stop",
			register: func() {
				agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.Agent != agenthooks.AgentCodex {
						t.Fatalf("agent: got %q want codex", e.Agent)
					}
					return agenthooks.Interrupt("run tests")
				})
			},
			stdin:      `{"session_id":"s-1","hook_event_name":"Stop"}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"run tests"`},
		},
		{
			name:  "codex-pre-tool-use deny",
			route: "codex-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					if e.Agent != agenthooks.AgentCodex || !e.IsShell() || e.Command != "rm -rf /" {
						t.Fatalf("unexpected event: %+v", e)
					}
					return agenthooks.Deny("blocked")
				})
			},
			stdin: `{
				"session_id":"s-2","cwd":"/repo",
				"tool_name":"Bash","tool_input":{"command":"rm -rf /"}
			}`,
			wantStdout: []string{`"permissionDecision":"deny"`, `"permissionDecisionReason":"blocked"`},
		},
		{
			name:  "codex-pre-tool-use ask collapses to deny (no ask channel documented)",
			route: "codex-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					return agenthooks.AskUser("confirm?")
				})
			},
			stdin: `{
				"session_id":"s-3","cwd":"/repo",
				"tool_name":"Bash","tool_input":{"command":"ls"}
			}`,
			wantStdout: []string{`"permissionDecision":"deny"`, `"confirm?"`},
			notStdout:  []string{`"ask"`},
		},
		{
			name:  "codex-pre-tool-use rewrite uses updatedInput",
			route: "codex-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					return agenthooks.AllowWithInput(json.RawMessage(`{"command":"ls -la"}`))
				})
			},
			stdin: `{
				"session_id":"s-4",
				"tool_name":"Bash","tool_input":{"command":"ls"}
			}`,
			wantStdout: []string{`"permissionDecision":"allow"`, `"updatedInput"`, `"ls -la"`},
		},
		{
			// A bare allow (no note/context/rewrite) writes nothing to stdout: the
			// live Codex CLI rejects permissionDecision:"allow" as unsupported even
			// though the published doc documents it — see runPreToolUse.
			name:  "codex-pre-tool-use plain allow has empty stdout",
			route: "codex-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					return agenthooks.Allow()
				})
			},
			stdin: `{
				"session_id":"s-4b",
				"tool_name":"Bash","tool_input":{"command":"ls"}
			}`,
			wantEmpty: true,
		},
		{
			name:  "codex-pre-file-write reject with context on apply_patch",
			route: "codex-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					if e.Agent != agenthooks.AgentCodex {
						t.Fatalf("agent: got %q want codex", e.Agent)
					}
					return agenthooks.RejectEditWithContext("secret detected", "remove it and retry")
				})
			},
			stdin: `{
				"session_id":"s-5","cwd":"/repo",
				"tool_name":"apply_patch","tool_input":{"input":"*** Begin Patch\n*** End Patch"}
			}`,
			wantStdout: []string{`"permissionDecision":"deny"`, `"secret detected"`, `"remove it and retry"`},
		},
		{
			// A plain approve writes nothing to stdout: the live Codex CLI rejects
			// hookSpecificOutput.permissionDecision:"allow" as an unsupported
			// PreToolUse decision, even though the published doc documents it —
			// see runPreToolUse in codex/adapters.go. Per that same doc, "Exit 0
			// with no output is treated as success and Codex continues".
			name:  "codex-pre-file-write passes through non-write tool with empty stdout",
			route: "codex-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					t.Fatal("handler should not fire for Bash")
					return agenthooks.AcceptEdit()
				})
			},
			stdin: `{
				"session_id":"s-6",
				"tool_name":"Bash","tool_input":{"command":"ls"}
			}`,
			wantEmpty: true,
		},
		{
			name:  "codex-after-file-write annotate",
			route: "codex-after-file-write",
			register: func() {
				agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
					if e.Agent != agenthooks.AgentCodex {
						t.Fatalf("agent: got %q want codex", e.Agent)
					}
					return agenthooks.AnnotateWrite("run go vet")
				})
			},
			stdin: `{
				"session_id":"s-7","cwd":"/repo",
				"tool_name":"apply_patch","tool_input":{"input":"patch"},"tool_response":{}
			}`,
			wantStdout: []string{`"additionalContext":"run go vet"`},
		},
		{
			name:  "codex-after-file-write skips non-write tool",
			route: "codex-after-file-write",
			register: func() {
				agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
					t.Fatal("handler should not fire for Bash")
					return agenthooks.AcceptWrite()
				})
			},
			stdin: `{
				"session_id":"s-8",
				"tool_name":"Bash","tool_input":{"command":"ls"},"tool_response":{}
			}`,
			wantStdout: []string{},
		},
		{
			name:  "codex-user-prompt-submit reject",
			route: "codex-user-prompt-submit",
			register: func() {
				agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
					if e.Agent != agenthooks.AgentCodex {
						t.Fatalf("agent: got %q want codex", e.Agent)
					}
					return agenthooks.RejectPrompt("no secrets")
				})
			},
			stdin:      `{"session_id":"s-9","hook_event_name":"UserPromptSubmit","prompt":"leak API_KEY=abc"}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"no secrets"`},
		},
		{
			name:  "codex-subagent-stop interrupt",
			route: "codex-subagent-stop",
			register: func() {
				agenthooks.WhenSubagentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					return agenthooks.Interrupt("finish subtask")
				})
			},
			stdin:      `{"session_id":"s-10","agent_id":"a-1"}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"finish subtask"`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agenthooks.ClearRoutes()
			tc.register()

			stdoutBuf := pipeStdio(t, tc.stdin)

			origArgs := os.Args
			os.Args = []string{"codexhook", tc.route}
			defer func() { os.Args = origArgs }()

			agenthooks.Dispatch()

			out := stdoutBuf()
			if tc.wantEmpty {
				if strings.TrimSpace(out) != "" {
					t.Fatalf("expected empty stdout for a plain approve, got: %q", out)
				}
				return
			}
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
					t.Fatalf("stdout unexpectedly contains %q\nfull output: %s", not, out)
				}
			}
		})
	}
}

func TestAgentCodexIDIsRegistered(t *testing.T) {
	if agenthooks.AgentCodex == "" {
		t.Fatal("AgentCodex must be defined")
	}
	for _, other := range []agenthooks.AgentID{
		agenthooks.AgentClaude,
		agenthooks.AgentCursor,
		agenthooks.AgentWindsurf,
		agenthooks.AgentDroid,
		agenthooks.AgentGemini,
		agenthooks.AgentCopilot,
		agenthooks.AgentCopilotCLI,
	} {
		if agenthooks.AgentCodex == other {
			t.Fatalf("AgentCodex collides with %q", other)
		}
	}
}

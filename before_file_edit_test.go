package agenthooks_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
)

// TestBeforeFileEditEndToEnd drives the pre-file-write GATE through Dispatch. This
// is the route a security scanner (e.g. the cx-security plugin's claude-pre-file-write)
// uses to BLOCK a vulnerable write before it lands — verified here end to end:
// deny + additionalContext on Claude, deny-only on Droid (no context channel), the
// allow/ask paths, and the non-write guard.
func TestBeforeFileEditEndToEnd(t *testing.T) {
	cases := []struct {
		name       string
		route      string
		register   func()
		stdin      string
		wantStdout []string
		notStdout  []string
	}{
		{
			name:  "claude-pre-file-write blocks vulnerable write with remediation context",
			route: "claude-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					if e.Agent != agenthooks.AgentClaude || e.FilePath != "/r/app.py" {
						t.Fatalf("unexpected event: agent=%q path=%q", e.Agent, e.FilePath)
					}
					if len(e.Changes) == 0 || !strings.Contains(e.Changes[0].After, "eval(") {
						t.Fatalf("expected proposed content to be visible pre-write, got %+v", e.Changes)
					}
					return agenthooks.RejectEditWithContext(
						"ASCA found a code-injection finding in /r/app.py",
						"Run /cx-security-asca to remediate, then retry.",
					)
				})
			},
			stdin: `{"session_id":"s","cwd":"/r","tool_name":"Write","tool_input":{"file_path":"/r/app.py","content":"eval(user_input)"}}`,
			wantStdout: []string{
				`"permissionDecision":"deny"`,
				`"permissionDecisionReason":"ASCA found a code-injection finding in /r/app.py"`,
				`"additionalContext":"Run /cx-security-asca to remediate, then retry."`,
			},
		},
		{
			name:  "claude-pre-file-write allows a clean write",
			route: "claude-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					return agenthooks.AcceptEdit()
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","tool_name":"Edit","tool_input":{"file_path":"/r/ok.go","old_string":"a","new_string":"b"}}`,
			wantStdout: []string{`"permissionDecision":"allow"`},
		},
		{
			name:  "claude-pre-file-write can ask for confirmation",
			route: "claude-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					return agenthooks.AskBeforeEdit("confirm this edit?")
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","tool_name":"Write","tool_input":{"file_path":"/r/x.go","content":"package main"}}`,
			wantStdout: []string{`"permissionDecision":"ask"`, `"confirm this edit?"`},
		},
		{
			name:  "claude-pre-file-write approves non-write tools without invoking the handler",
			route: "claude-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					t.Fatalf("handler must not run for a non-write tool (Bash)")
					return agenthooks.AcceptEdit()
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","tool_name":"Bash","tool_input":{"command":"ls"}}`,
			wantStdout: []string{`"permissionDecision":"allow"`},
		},
		{
			name:  "droid-pre-file-write blocks but drops context (no channel for it)",
			route: "droid-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					if e.Agent != agenthooks.AgentDroid {
						t.Fatalf("agent: %q", e.Agent)
					}
					return agenthooks.RejectEditWithContext("blocked by policy", "should be dropped on droid")
				})
			},
			stdin:      `{"session_id":"s","cwd":"/r","tool_name":"Create","tool_input":{"file_path":"/r/m.go","content":"x"}}`,
			wantStdout: []string{`"permissionDecision":"deny"`, `"blocked by policy"`},
			notStdout:  []string{"additionalContext", "should be dropped on droid"},
		},
		{
			name:  "gemini-before-file-tool blocks a write and folds remediation context into reason",
			route: "gemini-before-file-tool",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					if e.Agent != agenthooks.AgentGemini || e.FilePath != "/r/x.py" {
						t.Fatalf("unexpected event: agent=%q path=%q", e.Agent, e.FilePath)
					}
					if len(e.Changes) == 0 || !strings.Contains(e.Changes[0].After, "eval(") {
						t.Fatalf("expected proposed content pre-write, got %+v", e.Changes)
					}
					return agenthooks.RejectEditWithContext("ASCA finding in /r/x.py", "Run /cx-security-asca to remediate, then retry.")
				})
			},
			stdin: `{"session_id":"s","tool_name":"write_file","tool_input":{"file_path":"/r/x.py","content":"eval(x)"}}`,
			wantStdout: []string{
				`"decision":"deny"`,
				`ASCA finding in /r/x.py`,
				`Run /cx-security-asca to remediate, then retry.`,
			},
			notStdout: []string{"additionalContext"},
		},
		{
			name:  "gemini-before-file-tool approves non-write tools",
			route: "gemini-before-file-tool",
			register: func() {
				agenthooks.BeforeFileEdit(func(agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					t.Fatalf("handler must not run for a non-write tool (run_shell_command)")
					return agenthooks.AcceptEdit()
				})
			},
			stdin:      `{"session_id":"s","tool_name":"run_shell_command","tool_input":{"command":"ls"}}`,
			wantStdout: []string{}, // gemini ApproveToolCall() is an empty result ({})
		},
		{
			name:  "copilot-pre-file-write blocks with remediation context (PreToolUse)",
			route: "copilot-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					if e.Agent != agenthooks.AgentCopilot || e.FilePath != "/r/x.ts" {
						t.Fatalf("unexpected event: agent=%q path=%q", e.Agent, e.FilePath)
					}
					return agenthooks.RejectEditWithContext("ASCA finding", "Run /cx-security-asca then retry.")
				})
			},
			stdin: `{"tool_name":"createFile","tool_input":{"filePath":"/r/x.ts","content":"eval(x)"}}`,
			wantStdout: []string{
				`"permissionDecision":"deny"`,
				`"additionalContext":"Run /cx-security-asca then retry."`,
			},
		},
		{
			name:  "copilot-cli-pre-file-write blocks and delivers remediation context via additionalContext + folded reason (flat preToolUse)",
			route: "copilot-cli-pre-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					if e.Agent != agenthooks.AgentCopilotCLI {
						t.Fatalf("agent: %q", e.Agent)
					}
					return agenthooks.RejectEditWithContext("blocked by policy", "run the remediation skill then retry")
				})
			},
			stdin: `{"session_id":"s","tool_name":"create","tool_input":{"file_path":"/r/x.py","content":"eval(x)"}}`,
			// copilot-cli is FLAT: top-level permissionDecision (no hookSpecificOutput wrapper).
			// Context is emitted as additionalContext (forward-compat) AND folded into
			// permissionDecisionReason, which the CLI forwards today (github/copilot-cli#2585).
			wantStdout: []string{`"permissionDecision":"deny"`, `"additionalContext"`, `blocked by policy`, `run the remediation skill then retry`},
			notStdout:  []string{"hookSpecificOutput"},
		},
		{
			name:  "cursor-before-file-write blocks a Write via generic preToolUse",
			route: "cursor-before-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					if e.Agent != agenthooks.AgentCursor || e.FilePath != "/r/app.py" {
						t.Fatalf("unexpected event: agent=%q path=%q", e.Agent, e.FilePath)
					}
					return agenthooks.RejectEditWithContext("ASCA finding", "context dropped on cursor")
				})
			},
			stdin:      `{"conversation_id":"c","tool_name":"Write","tool_input":{"file_path":"/r/app.py","content":"eval(x)"},"cwd":"/r"}`,
			wantStdout: []string{`"permission":"deny"`, `ASCA finding`},
			notStdout:  []string{"additional_context", "context dropped on cursor"},
		},
		{
			name:  "cursor-before-file-write approves non-Write tools",
			route: "cursor-before-file-write",
			register: func() {
				agenthooks.BeforeFileEdit(func(agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					t.Fatalf("handler must not run for a non-Write tool (Read)")
					return agenthooks.AcceptEdit()
				})
			},
			stdin:      `{"conversation_id":"c","tool_name":"Read","tool_input":{"file_path":"/r/app.py"},"cwd":"/r"}`,
			wantStdout: []string{`"permission":"allow"`},
		},
		{
			name:  "cursor-before-file-read gates a read via BeforeFileEdit (read-as-edit, ast-cli model)",
			route: "cursor-before-file-read",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					// ast-cli's model: Cursor reads arrive with empty Changes; scan the path.
					if e.Agent != agenthooks.AgentCursor || len(e.Changes) != 0 || e.FilePath != "/r/secret.env" {
						t.Fatalf("expected a Cursor read event for /r/secret.env, got agent=%q path=%q changes=%d", e.Agent, e.FilePath, len(e.Changes))
					}
					return agenthooks.RejectEdit("secret detected in /r/secret.env")
				})
			},
			stdin:      `{"conversation_id":"c","file_path":"/r/secret.env","content":"API_KEY=xxx"}`,
			wantStdout: []string{`"permission":"deny"`, `secret detected in /r/secret.env`},
		},
		{
			name:  "windsurf-pre-write-code allows a clean write (pre_write_code gate)",
			route: "windsurf-pre-write-code",
			register: func() {
				agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
					if e.Agent != agenthooks.AgentWindsurf || e.FilePath != "/r/app.go" {
						t.Fatalf("unexpected event: agent=%q path=%q", e.Agent, e.FilePath)
					}
					if len(e.Changes) == 0 || e.Changes[0].After != "b" {
						t.Fatalf("expected proposed edit content, got %+v", e.Changes)
					}
					return agenthooks.AcceptEdit()
				})
			},
			stdin:      `{"trajectory_id":"t","tool_info":{"file_path":"/r/app.go","edits":[{"old_string":"a","new_string":"b"}]}}`,
			wantStdout: []string{}, // windsurf AllowWrite() is an empty result ({}); deny blocks via exit 2 (RunE)
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
			for _, notWant := range tc.notStdout {
				if strings.Contains(out, notWant) {
					t.Fatalf("stdout should not contain %q\nfull output: %s", notWant, out)
				}
			}
		})
	}
}

// TestBeforeFileEditFailsClosed verifies the gate's fallback: if the route fires
// with no handler registered, it denies (fails closed) rather than silently
// permitting the write.
func TestBeforeFileEditFailsClosed(t *testing.T) {
	agenthooks.ClearRoutes()
	// Register a different hook so the binary has routes, but NOT BeforeFileEdit's
	// default — then force the claude-pre-file-write route to dispatch.
	agenthooks.BeforeFileEditScenario("unused", func(agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
		return agenthooks.AcceptEdit()
	})

	stdoutBuf := pipeStdio(t, `{"session_id":"s","cwd":"/r","tool_name":"Write","tool_input":{"file_path":"/r/x.go","content":"x"}}`)
	origArgs := os.Args
	os.Args = []string{"hook", "claude-pre-file-write"} // no scenario key → no default → fallback
	defer func() { os.Args = origArgs }()

	agenthooks.Dispatch()

	out := stdoutBuf()
	if !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("expected fail-closed deny, got: %s", out)
	}
}

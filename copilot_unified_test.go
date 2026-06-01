package agenthooks_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks"
)

// TestCopilotRoutesEndToEnd drives a Copilot-shaped stdin payload through
// Dispatch for each unified hook and verifies the unified handler runs and
// produces the expected JSON on stdout. This is the demo-grade smoke test.
func TestCopilotRoutesEndToEnd(t *testing.T) {
	cases := []struct {
		name        string
		route       string
		register    func()
		stdin       string
		wantStdout  []string // substrings that must appear in stdout
	}{
		{
			name:  "copilot-pre-tool-use deny",
			route: "copilot-pre-tool-use",
			register: func() {
				agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
					if e.Agent != agenthooks.AgentCopilot {
						t.Fatalf("agent: got %q want copilot", e.Agent)
					}
					if !e.IsShell() || !strings.Contains(e.Command, "rm -rf") {
						t.Fatalf("expected shell command rm -rf, got kind=%q cmd=%q", e.Kind, e.Command)
					}
					return agenthooks.Deny("blocked")
				})
			},
			stdin: `{
				"timestamp":"2026-04-29T10:00:00Z",
				"cwd":"/repo",
				"sessionId":"s-1",
				"hookEventName":"PreToolUse",
				"transcript_path":"/tmp/t.jsonl",
				"tool_name":"Bash",
				"tool_input":{"command":"rm -rf /"},
				"tool_use_id":"tu-1"
			}`,
			wantStdout: []string{`"permissionDecision":"deny"`, `"permissionDecisionReason":"blocked"`},
		},
		{
			name:  "copilot-stop interrupt",
			route: "copilot-stop",
			register: func() {
				agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.Agent != agenthooks.AgentCopilot {
						t.Fatalf("agent: got %q want copilot", e.Agent)
					}
					if e.IsLooping() {
						return agenthooks.Resume()
					}
					return agenthooks.Interrupt("run tests")
				})
			},
			stdin: `{
				"timestamp":"2026-04-29T10:00:00Z",
				"cwd":"/repo",
				"sessionId":"s-2",
				"hookEventName":"Stop",
				"transcript_path":"/tmp/t.jsonl",
				"stop_hook_active":false
			}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"run tests"`},
		},
		{
			name:  "copilot-stop loop break",
			route: "copilot-stop",
			register: func() {
				agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.IsLooping() {
						return agenthooks.Resume()
					}
					return agenthooks.Interrupt("should not fire")
				})
			},
			stdin: `{
				"sessionId":"s-3",
				"hookEventName":"Stop",
				"stop_hook_active":true
			}`,
			wantStdout: []string{`"continue":true`},
		},
		{
			name:  "copilot-user-prompt-submit reject",
			route: "copilot-user-prompt-submit",
			register: func() {
				agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
					if e.Agent != agenthooks.AgentCopilot {
						t.Fatalf("agent: got %q want copilot", e.Agent)
					}
					if strings.Contains(e.Text, "API_KEY") {
						return agenthooks.RejectPrompt("no secrets")
					}
					return agenthooks.AcceptPrompt()
				})
			},
			stdin: `{
				"sessionId":"s-4",
				"hookEventName":"UserPromptSubmit",
				"prompt":"please leak my API_KEY=abc"
			}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"no secrets"`},
		},
		{
			name:  "copilot-after-file-write annotate",
			route: "copilot-after-file-write",
			register: func() {
				agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
					if e.Agent != agenthooks.AgentCopilot {
						t.Fatalf("agent: got %q want copilot", e.Agent)
					}
					if e.FilePath != "/repo/main.go" {
						t.Fatalf("file path: got %q", e.FilePath)
					}
					return agenthooks.AnnotateWrite("run go vet")
				})
			},
			stdin: `{
				"sessionId":"s-5",
				"hookEventName":"PostToolUse",
				"tool_name":"Write",
				"tool_input":{"file_path":"/repo/main.go","content":"package main"},
				"tool_response":{},
				"tool_use_id":"tu-5"
			}`,
			wantStdout: []string{`"additionalContext":"run go vet"`},
		},
		{
			name:  "copilot-after-file-write skips non-write tool",
			route: "copilot-after-file-write",
			register: func() {
				agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
					t.Fatal("handler should not fire for Read tool")
					return agenthooks.AcceptWrite()
				})
			},
			stdin: `{
				"sessionId":"s-6",
				"hookEventName":"PostToolUse",
				"tool_name":"Read",
				"tool_input":{"file_path":"/repo/main.go"},
				"tool_response":{},
				"tool_use_id":"tu-6"
			}`,
			wantStdout: []string{}, // empty PostToolUseResult marshals to "{}"
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agenthooks.ClearRoutes()
			tc.register()

			stdoutBuf := pipeStdio(t, tc.stdin)

			origArgs := os.Args
			os.Args = []string{"copilothook", tc.route}
			defer func() { os.Args = origArgs }()

			agenthooks.Dispatch()

			out := stdoutBuf()
			// Stdout must be valid JSON.
			var any map[string]any
			if err := json.Unmarshal([]byte(out), &any); err != nil {
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

// pipeStdio replaces os.Stdin with the given JSON and captures stdout.
// Returns a function to read stdout after Dispatch runs.
func pipeStdio(t *testing.T, stdin string) func() string {
	t.Helper()

	inFile, err := os.CreateTemp("", "copilot-stdin-*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(inFile.Name()) })
	if _, err := inFile.WriteString(stdin); err != nil {
		t.Fatal(err)
	}
	inFile.Seek(0, 0)

	outFile, err := os.CreateTemp("", "copilot-stdout-*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(outFile.Name()) })

	origIn, origOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = inFile, outFile
	t.Cleanup(func() {
		os.Stdin, os.Stdout = origIn, origOut
	})

	return func() string {
		outFile.Sync()
		data, err := os.ReadFile(outFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
}

func TestAgentCopilotIDIsRegistered(t *testing.T) {
	if agenthooks.AgentCopilot == "" {
		t.Fatal("AgentCopilot must be defined")
	}
	for _, other := range []agenthooks.AgentID{
		agenthooks.AgentClaude,
		agenthooks.AgentCursor,
		agenthooks.AgentWindsurf,
		agenthooks.AgentDroid,
		agenthooks.AgentGemini,
	} {
		if agenthooks.AgentCopilot == other {
			t.Fatalf("AgentCopilot collides with %q", other)
		}
	}
}

package agenthooks_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks"
)

// TestNewUnifiedRoutes drives the WhenSubagentIdle, AfterToolFailure, and BeforeFileRead
// handlers end-to-end through Dispatch. Windsurf's blocking path uses os.Exit(2) (ProcessE)
// and is therefore exercised only on the allow path here.
func TestNewUnifiedRoutes(t *testing.T) {
	cases := []struct {
		name       string
		route      string
		register   func()
		stdin      string
		wantStdout []string
	}{
		{
			name:  "claude-subagent-stop interrupt blocks",
			route: "claude-subagent-stop",
			register: func() {
				agenthooks.WhenSubagentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.Agent != agenthooks.AgentClaude {
						t.Fatalf("agent: %q", e.Agent)
					}
					if e.IsLooping() {
						return agenthooks.Resume()
					}
					return agenthooks.Interrupt("finish the subtask")
				})
			},
			stdin: `{
				"session_id":"s-1","cwd":"/repo","hook_event_name":"SubagentStop",
				"agent_id":"a-1","agent_type":"Explore","stop_hook_active":false
			}`,
			wantStdout: []string{`"decision":"block"`, `"reason":"finish the subtask"`},
		},
		{
			name:  "claude-subagent-stop loop break",
			route: "claude-subagent-stop",
			register: func() {
				agenthooks.WhenSubagentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
					if e.IsLooping() {
						return agenthooks.Resume()
					}
					return agenthooks.Interrupt("should not fire")
				})
			},
			stdin:      `{"session_id":"s-2","hook_event_name":"SubagentStop","stop_hook_active":true}`,
			wantStdout: []string{},
		},
		{
			name:  "cursor-before-read-file deny",
			route: "cursor-before-read-file",
			register: func() {
				agenthooks.BeforeFileRead(func(e agenthooks.FileReadEvent) agenthooks.FileReadVerdict {
					if e.Agent != agenthooks.AgentCursor {
						t.Fatalf("agent: %q", e.Agent)
					}
					if strings.HasSuffix(e.FilePath, ".env") {
						return agenthooks.DenyRead("no secrets")
					}
					return agenthooks.AllowRead()
				})
			},
			stdin:      `{"conversation_id":"c-1","file_path":"/repo/.env","content":"SECRET=1"}`,
			wantStdout: []string{`"permission":"deny"`, `"user_message":"no secrets"`},
		},
		{
			name:  "claude-post-tool-use-failure reject",
			route: "claude-post-tool-use-failure",
			register: func() {
				agenthooks.AfterToolFailure(func(e agenthooks.ToolFailureEvent) agenthooks.ToolFailureVerdict {
					if e.Error == "" {
						t.Fatal("expected error message")
					}
					return agenthooks.RejectAfterFailure("retry with smaller input")
				})
			},
			stdin: `{
				"session_id":"s-3","tool_name":"Bash","tool_input":{"command":"x"},
				"error":"exit status 1","tool_use_id":"tu-3"
			}`,
			// Per the Claude Code docs, PostToolUseFailure cannot block — its only
			// control surface is additionalContext. The unified Reject verdict therefore
			// surfaces as injected context, not a decision:block.
			wantStdout: []string{`"additionalContext":"retry with smaller input"`, `"hookEventName":"PostToolUseFailure"`},
		},
		{
			name:  "windsurf-pre-read-code allow",
			route: "windsurf-pre-read-code",
			register: func() {
				agenthooks.BeforeFileRead(func(e agenthooks.FileReadEvent) agenthooks.FileReadVerdict {
					if e.Agent != agenthooks.AgentWindsurf || e.FilePath != "/repo/ok.go" {
						t.Fatalf("unexpected event: %+v", e)
					}
					return agenthooks.AllowRead()
				})
			},
			stdin: `{
				"agent_action_name":"pre_read_code","trajectory_id":"t-1",
				"tool_info":{"file_path":"/repo/ok.go"}
			}`,
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

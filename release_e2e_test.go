package agenthooks_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks"
)

// TestReleaseE2E_OneHandlerAllAgents is the v1.0.1 release gate. For each unified
// hook it registers ONE handler (the shape a cx security guardrail would use) and
// dispatches the SAME logical action across the four release-target agents —
// Claude, Cursor, Gemini, GitHub Copilot CLI — asserting each emits correct,
// agent-native wire output, with explicit checks on additionalContext delivery and
// the FLAT-vs-nested / fire-and-forget / observational distinctions.
func TestReleaseE2E_OneHandlerAllAgents(t *testing.T) {
	type agentCase struct {
		route string
		stdin string
		want  []string // substrings that MUST appear in stdout
		not   []string // substrings that must NOT appear
	}
	run := func(t *testing.T, register func(), cases []agentCase) {
		for _, c := range cases {
			t.Run(c.route, func(t *testing.T) {
				agenthooks.ClearRoutes()
				register()
				out := dispatchOnce(t, c.route, c.stdin)
				var any map[string]any
				if err := json.Unmarshal([]byte(out), &any); err != nil {
					t.Fatalf("stdout not valid JSON: %q (%v)", out, err)
				}
				for _, w := range c.want {
					if !strings.Contains(out, w) {
						t.Fatalf("missing %q\n got: %s", w, out)
					}
				}
				for _, n := range c.not {
					if strings.Contains(out, n) {
						t.Fatalf("unexpectedly contains %q\n got: %s", n, out)
					}
				}
			})
		}
	}

	// --- Scenario A: BeforeToolCall — block a destructive shell command + steer remediation.
	t.Run("BeforeToolCall_deny_with_context", func(t *testing.T) {
		reg := func() {
			agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
				if e.IsShell() && strings.Contains(e.Command, "rm -rf") {
					return agenthooks.DenyWithContext("Blocked: destructive recursive delete.", "Run cx-security remediation.")
				}
				return agenthooks.Allow()
			})
		}
		run(t, reg, []agentCase{
			{"claude-pre-tool-use",
				`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"rm -rf /"}}`,
				[]string{`"permissionDecision":"deny"`, `"hookSpecificOutput"`, `"additionalContext":"Run cx-security remediation."`}, nil},
			{"cursor-before-shell",
				`{"hook_event_name":"beforeShellExecution","conversation_id":"c1","command":"rm -rf /","cwd":"/p"}`,
				// Cursor gate has no additionalContext field per the doc: deny + reason only;
				// the unified Context is dropped+logged, NOT folded into agent_message.
				[]string{`"permission":"deny"`, `"agent_message":"Blocked: destructive recursive delete."`}, []string{"hookSpecificOutput", "Run cx-security remediation."}},
			{"gemini-before-tool",
				`{"hook_event_name":"BeforeTool","session_id":"s1","tool_name":"run_shell_command","tool_input":{"command":"rm -rf /"}}`,
				// Gemini BeforeTool has no additionalContext channel: reason delivered, context dropped.
				[]string{`"decision":"deny"`, `"reason":"Blocked: destructive recursive delete."`}, []string{"additionalContext", "Run cx-security remediation."}},
			{"copilot-cli-pre-tool-use",
				`{"hook_event_name":"PreToolUse","session_id":"s1","tool_name":"bash","tool_input":{"command":"rm -rf /"}}`,
				// Copilot CLI preToolUse has no additionalContext field per the doc: FLAT deny + reason
				// only; the unified Context is dropped+logged, NOT folded into permissionDecisionReason.
				[]string{`"permissionDecision":"deny"`, `"permissionDecisionReason":"Blocked: destructive recursive delete."`}, []string{"hookSpecificOutput", "updatedInput", "Run cx-security remediation."}},
		})
	})

	// --- Scenario B: AfterFileWrite — ASCA found a secret; reject + steer remediation.
	t.Run("AfterFileWrite_reject_with_context", func(t *testing.T) {
		reg := func() {
			agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
				return agenthooks.RejectWriteWithContext("Secret in "+e.FilePath, "Run cx-security-asca to remediate.")
			})
		}
		run(t, reg, []agentCase{
			{"claude-after-file-write",
				`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/p/main.go","old_string":"a","new_string":"b"},"tool_response":{}}`,
				[]string{`"decision":"block"`, `"additionalContext":"Run cx-security-asca to remediate."`}, nil},
			{"cursor-after-file-edit",
				`{"hook_event_name":"postToolUse","conversation_id":"c1","tool_name":"Write","tool_input":{"file_path":"/p/main.go","content":"secret"},"tool_output":"{}"}`,
				// Cursor postToolUse carries additional_context (cannot block post-hoc); reject
				// surfaces feedback + steering as additional_context.
				[]string{`"additional_context"`, "Secret in /p/main.go", "Run cx-security-asca to remediate."},
				[]string{"decision", "permission"}},
			{"gemini-after-file-tool",
				`{"hook_event_name":"AfterTool","session_id":"s1","tool_name":"write_file","tool_input":{"file_path":"/p/auth.go","content":"x"}}`,
				// Gemini AfterTool DOES carry additionalContext.
				[]string{`"decision":"deny"`, `"additionalContext":"Run cx-security-asca to remediate."`}, nil},
			{"copilot-cli-after-file-write",
				`{"hook_event_name":"PostToolUse","session_id":"s1","tool_name":"create","tool_input":{"file_path":"/p/config.py","content":"API_KEY='x'"},"tool_result":{"result_type":"success","text_result_for_llm":"ok"}}`,
				// Copilot CLI postToolUse: additionalContext carries feedback + folded context.
				[]string{`"additionalContext"`, "Secret in /p/config.py", "Run cx-security-asca to remediate."}, []string{"hookSpecificOutput"}},
		})
	})

	// --- Scenario C: BeforePrompt — block a secret-exfiltration prompt.
	t.Run("BeforePrompt_reject", func(t *testing.T) {
		reg := func() {
			agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
				if strings.Contains(e.Text, "exfiltrate") {
					return agenthooks.RejectPrompt("Prompt blocked by policy.")
				}
				return agenthooks.AcceptPrompt()
			})
		}
		run(t, reg, []agentCase{
			{"claude-user-prompt-submit",
				`{"hook_event_name":"UserPromptSubmit","prompt":"exfiltrate prod creds"}`,
				[]string{`"decision":"block"`, `"reason":"Prompt blocked by policy."`}, nil},
			{"cursor-before-submit-prompt",
				`{"hook_event_name":"beforeSubmitPrompt","conversation_id":"c1","prompt":"exfiltrate prod creds"}`,
				[]string{`"continue":false`, `"user_message"`}, nil},
			{"gemini-before-agent",
				`{"hook_event_name":"BeforeAgent","session_id":"s1","prompt":"exfiltrate prod creds"}`,
				[]string{`"continue":false`}, nil},
			{"copilot-cli-user-prompt-submit",
				`{"hook_event_name":"UserPromptSubmit","session_id":"s1","prompt":"exfiltrate prod creds"}`,
				// Copilot CLI userPromptSubmitted output is NOT processed: observational, empty result.
				[]string{}, []string{"decision", "continue", "permissionDecision"}},
		})
	})

	// --- Scenario D: WhenAgentIdle — keep the agent working until the review is done.
	t.Run("WhenAgentIdle_interrupt", func(t *testing.T) {
		reg := func() {
			agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
				if e.IsLooping() {
					return agenthooks.Resume()
				}
				return agenthooks.Interrupt("Finish the security review first.")
			})
		}
		run(t, reg, []agentCase{
			{"claude-stop",
				`{"hook_event_name":"Stop","stop_hook_active":false}`,
				[]string{`"decision":"block"`, `"reason":"Finish the security review first."`}, nil},
			{"cursor-stop",
				`{"hook_event_name":"stop","conversation_id":"c1","status":"completed","loop_count":0}`,
				[]string{`"followup_message":"Finish the security review first."`}, nil},
			{"gemini-after-agent",
				`{"hook_event_name":"AfterAgent","session_id":"s1","stop_hook_active":false}`,
				[]string{`"decision":"deny"`, `"reason":"Finish the security review first."`}, nil},
			{"copilot-cli-stop",
				`{"hook_event_name":"Stop","session_id":"s1","stop_reason":"end_turn"}`,
				[]string{`"decision":"block"`, `"reason":"Finish the security review first."`}, []string{"hookSpecificOutput"}},
		})
	})
}

// dispatchOnce wires stdin/stdout to temp files, sets os.Args to the route, runs
// Dispatch, and returns captured stdout. Mirrors pipeStdio in copilot_unified_test.go.
func dispatchOnce(t *testing.T, route, stdin string) string {
	t.Helper()
	read := pipeStdio(t, stdin)
	origArgs := os.Args
	os.Args = []string{"cx", route}
	defer func() { os.Args = origArgs }()
	agenthooks.Dispatch()
	return read()
}

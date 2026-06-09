package copilotcli

import (
	"encoding/json"
	"strings"
	"testing"
)

func marshal(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

// TestBuilderShapes pins the exact FLAT JSON each Copilot CLI response builder
// produces. The CLI's defining trait vs. the VS Code copilot package is that
// output is FLAT — no hookSpecificOutput wrapper, modifiedArgs (not updatedInput),
// top-level decision — so these assertions are the regression guard for that.
func TestBuilderShapes(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		// agentStop / subagentStop — top-level decision/reason
		{"LetStop", marshal(t, LetStop()), `{}`},
		{"HaltAndContinue", marshal(t, HaltAndContinue("keep going")), `{"decision":"block","reason":"keep going"}`},
		{"LetSubagentStop", marshal(t, LetSubagentStop()), `{}`},
		{"KeepSubagentRunning", marshal(t, KeepSubagentRunning("not done")), `{"decision":"block","reason":"not done"}`},

		// preToolUse — flat permissionDecision + modifiedArgs
		{"ApproveToolUse", marshal(t, ApproveToolUse()), `{"permissionDecision":"allow"}`},
		{"ApproveToolUseWithNote", marshal(t, ApproveToolUseWithNote("ok")), `{"permissionDecision":"allow","permissionDecisionReason":"ok"}`},
		{"ApproveToolUseWithInput", marshal(t, ApproveToolUseWithInput(json.RawMessage(`{"command":"ls -la"}`))), `{"permissionDecision":"allow","modifiedArgs":{"command":"ls -la"}}`},
		{"DenyToolUse", marshal(t, DenyToolUse("blocked")), `{"permissionDecision":"deny","permissionDecisionReason":"blocked"}`},
		{"AskUserAboutTool", marshal(t, AskUserAboutTool("confirm?")), `{"permissionDecision":"ask","permissionDecisionReason":"confirm?"}`},

		// postToolUse — modifiedResult + additionalContext
		{"AcknowledgeToolUse", marshal(t, AcknowledgeToolUse()), `{}`},
		{"AddToolContext", marshal(t, AddToolContext("note")), `{"additionalContext":"note"}`},
		{"ReplaceToolResult", marshal(t, ReplaceToolResult("[redacted]")), `{"modifiedResult":{"resultType":"success","textResultForLlm":"[redacted]"}}`},
		{"RejectToolResult", marshal(t, RejectToolResult("retry")), `{"additionalContext":"retry"}`},

		// postToolUseFailure / userPromptSubmitted
		{"AcknowledgeFailure", marshal(t, AcknowledgeFailure()), `{}`},
		{"AnnotateFailure", marshal(t, AnnotateFailure("fix it")), `{"additionalContext":"fix it"}`},
		{"AcknowledgePrompt", marshal(t, AcknowledgePrompt()), `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("%s = %s, want %s", tc.name, tc.got, tc.want)
			}
			// FLAT invariant: the CLI must never emit the VS Code wrapper or its key.
			if strings.Contains(tc.got, "hookSpecificOutput") {
				t.Fatalf("%s emitted a hookSpecificOutput wrapper — CLI output must be flat: %s", tc.name, tc.got)
			}
			if strings.Contains(tc.got, "updatedInput") {
				t.Fatalf("%s used updatedInput — CLI uses modifiedArgs: %s", tc.name, tc.got)
			}
		})
	}
}

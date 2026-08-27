package codex

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

// TestBuilderShapes pins the exact JSON each Codex response builder produces,
// mirroring Claude's nested hookSpecificOutput shape. It also guards the
// documented absence of an "ask" decision value: no builder in this package
// should ever emit permissionDecision:"ask".
func TestBuilderShapes(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"LetStop", marshal(t, LetStop()), `{"continue":true}`},
		{"HaltAndContinue", marshal(t, HaltAndContinue("keep going")), `{"decision":"block","reason":"keep going"}`},
		{"LetSubagentStop", marshal(t, LetSubagentStop()), `{"continue":true}`},
		{"HaltSubagent", marshal(t, HaltSubagent("not done")), `{"decision":"block","reason":"not done"}`},

		{"ApproveToolUse", marshal(t, ApproveToolUse()), `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}`},
		{"ApproveToolUseWithNote", marshal(t, ApproveToolUseWithNote("ok")), `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"ok"}}`},
		{"ApproveToolUseWithInput", marshal(t, ApproveToolUseWithInput(json.RawMessage(`{"command":"ls -la"}`))), `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","updatedInput":{"command":"ls -la"}}}`},
		{"ApproveToolUseWithContext", marshal(t, ApproveToolUseWithContext("ctx")), `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","additionalContext":"ctx"}}`},
		{"DenyToolUse", marshal(t, DenyToolUse("blocked")), `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"blocked"}}`},
		{"DenyToolUseWithContext", marshal(t, DenyToolUseWithContext("blocked", "remediate")), `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"blocked","additionalContext":"remediate"}}`},

		{"AcknowledgeToolUse", marshal(t, AcknowledgeToolUse()), `{}`},
		{"AddToolContext", marshal(t, AddToolContext("note")), `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"note"}}`},
		{"RejectToolResult", marshal(t, RejectToolResult("retry")), `{"decision":"block","reason":"retry"}`},
		{"RejectToolResultWithContext", marshal(t, RejectToolResultWithContext("retry", "remediate")), `{"decision":"block","reason":"retry","hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"remediate"}}`},

		{"ApprovePrompt", marshal(t, ApprovePrompt()), `{"continue":true}`},
		{"RejectPrompt", marshal(t, RejectPrompt("no secrets")), `{"decision":"block","reason":"no secrets"}`},
		{"AppendToPrompt", marshal(t, AppendToPrompt("extra")), `{"continue":true,"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"extra"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("%s = %s, want %s", tc.name, tc.got, tc.want)
			}
			if strings.Contains(tc.got, `"ask"`) {
				t.Fatalf("%s emitted an ask decision — Codex's doc documents no ask value: %s", tc.name, tc.got)
			}
		})
	}
}

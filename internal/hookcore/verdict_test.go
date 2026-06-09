package hookcore

import (
	"encoding/json"
	"testing"
)

// TestToolVerdictPath locks the verdict-field precedence in one place. Before this
// classifier existed, the same precedence was re-implemented in the Claude, Copilot,
// and Droid adapters and could only be exercised through full Dispatch e2e tests;
// now a precedence regression is caught here once for all three.
func TestToolVerdictPath(t *testing.T) {
	raw := json.RawMessage(`{"x":1}`)
	cases := []struct {
		name string
		v    ToolVerdict
		want ToolPath
	}{
		{"plain allow", Allow(), PathAllow},
		{"allow with input", AllowWithInput(raw), PathAllowWithInput},
		{"allow with context", AllowWithContext("ctx"), PathAllowWithContext},
		{"allow with note", AllowWithNote("note"), PathAllowWithNote},
		{"ask", AskUser("confirm?"), PathAsk},
		{"plain deny", Deny("no"), PathDeny},
		{"deny with context", DenyWithContext("no", "remediate"), PathDenyWithContext},

		// precedence on allow: RewrittenInput > Context > Message
		{"input beats context+note", ToolVerdict{Permit: true, RewrittenInput: raw, Context: "c", Message: "m"}, PathAllowWithInput},
		{"context beats note", ToolVerdict{Permit: true, Context: "c", Message: "m"}, PathAllowWithContext},

		// precedence on deny: NeedsConfirm > Context
		{"confirm beats deny-context", ToolVerdict{Permit: false, NeedsConfirm: true, Context: "c", Message: "m"}, PathAsk},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.Path(); got != tc.want {
				t.Fatalf("Path() = %v, want %v", got, tc.want)
			}
		})
	}
}

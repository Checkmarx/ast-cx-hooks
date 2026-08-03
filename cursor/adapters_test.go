package cursor

import (
	"testing"

	"github.com/Checkmarx/ast-cx-hooks/internal/hookcore"
)

func TestAgentMessageFromVerdict(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		v    hookcore.ToolVerdict
		want string
	}{
		{"message only", hookcore.Deny("blocked"), "blocked"},
		{"context only", hookcore.ToolVerdict{Permit: false, Context: "run remediation"}, "run remediation"},
		{
			"message and context merged",
			hookcore.DenyWithContext("ASCA finding", "run remediation skill"),
			"ASCA finding\n\nrun remediation skill",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := agentMessageFromVerdict(tc.v); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestToolPreDecisionDenyWithContext(t *testing.T) {
	out := toolPreDecision(hookcore.DenyWithContext("finding", "remediation steps"))
	if out.Permission != "deny" {
		t.Fatalf("permission = %q", out.Permission)
	}
	if out.UserNote != "finding" {
		t.Fatalf("user_message = %q", out.UserNote)
	}
	wantAgent := DenyAgentMessagePrefix + "finding\n\nremediation steps"
	if out.AgentNote != wantAgent {
		t.Fatalf("agent_message = %q, want %q", out.AgentNote, wantAgent)
	}
}

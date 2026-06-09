package claude_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks/claude"
)

func TestStopResponses(t *testing.T) {
	r := claude.LetStop()
	if r.Proceed == nil || !*r.Proceed {
		t.Fatal("LetStop should set Proceed=true")
	}
	if r.Decision != "" {
		t.Fatal("LetStop should not set Decision")
	}

	b := claude.HaltAndContinue("keep working")
	if b.Decision != "block" {
		t.Fatalf("HaltAndContinue: Decision=%q, want block", b.Decision)
	}
	if b.Reason != "keep working" {
		t.Fatalf("HaltAndContinue: Reason=%q", b.Reason)
	}
}

func TestPreToolUseResponses(t *testing.T) {
	a := claude.ApproveToolUse()
	if a.Details == nil || a.Details.Decision != "allow" {
		t.Fatal("ApproveToolUse should set decision=allow")
	}

	d := claude.DenyToolUse("dangerous")
	if d.Details == nil || d.Details.Decision != "deny" {
		t.Fatal("DenyToolUse should set decision=deny")
	}
	if d.Details.DecisionReason != "dangerous" {
		t.Fatalf("DenyToolUse: DecisionReason=%q", d.Details.DecisionReason)
	}

	ask := claude.AskUserAboutTool("confirm?")
	if ask.Details == nil || ask.Details.Decision != "ask" {
		t.Fatal("AskUserAboutTool should set decision=ask")
	}
}

func TestUserPromptSubmitResponses(t *testing.T) {
	a := claude.ApprovePrompt()
	if a.Proceed == nil || !*a.Proceed {
		t.Fatal("ApprovePrompt should set Proceed=true")
	}

	r := claude.RejectPrompt("blocked")
	if r.Decision != "block" {
		t.Fatalf("RejectPrompt: Decision=%q, want block", r.Decision)
	}

	e := claude.AppendToPrompt("extra context")
	if e.Details == nil || e.Details.ExtraContext != "extra context" {
		t.Fatal("AppendToPrompt should set ExtraContext")
	}
}

func TestPostToolUseResponses(t *testing.T) {
	a := claude.AcknowledgeToolUse()
	if a.Decision != "" || a.Details != nil {
		t.Fatal("AcknowledgeToolUse should be empty")
	}

	c := claude.AddToolContext("note")
	if c.Details == nil || c.Details.ExtraContext != "note" {
		t.Fatal("AddToolContext should set ExtraContext")
	}

	r := claude.RejectToolResult("bad edit")
	if r.Decision != "block" {
		t.Fatalf("RejectToolResult: Decision=%q", r.Decision)
	}
}

// TestDenyToolUseWithContextJSON verifies that a PreToolUse deny-with-context
// serializes to JSON carrying BOTH the deny decision (+ reason) AND the
// additionalContext value.
func TestDenyToolUseWithContextJSON(t *testing.T) {
	r := claude.DenyToolUseWithContext("blocked: secret detected", "run the cx-remediation skill")
	if r.Details == nil {
		t.Fatal("DenyToolUseWithContext should set Details")
	}
	if r.Details.Decision != "deny" {
		t.Fatalf("Decision=%q, want deny", r.Details.Decision)
	}
	if r.Details.DecisionReason != "blocked: secret detected" {
		t.Fatalf("DecisionReason=%q", r.Details.DecisionReason)
	}
	if r.Details.ExtraContext != "run the cx-remediation skill" {
		t.Fatalf("ExtraContext=%q", r.Details.ExtraContext)
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := string(b)
	for _, want := range []string{
		`"permissionDecision":"deny"`,
		`"permissionDecisionReason":"blocked: secret detected"`,
		`"additionalContext":"run the cx-remediation skill"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("JSON %s missing %s", out, want)
		}
	}

	// Round-trip back into the wire struct.
	var rt claude.PreToolUseResult
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rt.Details == nil || rt.Details.Decision != "deny" || rt.Details.ExtraContext != "run the cx-remediation skill" {
		t.Fatalf("round-trip lost fields: %+v", rt.Details)
	}
}

// TestRejectToolResultWithContextJSON verifies that a PostToolUse
// reject-with-context serializes to JSON carrying BOTH the block decision
// (+ reason) AND the additionalContext value.
func TestRejectToolResultWithContextJSON(t *testing.T) {
	r := claude.RejectToolResultWithContext("write rejected: hardcoded credential", "run the cx-remediation skill on the finding")
	if r.Decision != "block" {
		t.Fatalf("Decision=%q, want block", r.Decision)
	}
	if r.Reason != "write rejected: hardcoded credential" {
		t.Fatalf("Reason=%q", r.Reason)
	}
	if r.Details == nil || r.Details.ExtraContext != "run the cx-remediation skill on the finding" {
		t.Fatalf("ExtraContext not set: %+v", r.Details)
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := string(b)
	for _, want := range []string{
		`"decision":"block"`,
		`"reason":"write rejected: hardcoded credential"`,
		`"additionalContext":"run the cx-remediation skill on the finding"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("JSON %s missing %s", out, want)
		}
	}

	var rt claude.PostToolUseResult
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rt.Decision != "block" || rt.Details == nil || rt.Details.ExtraContext != "run the cx-remediation skill on the finding" {
		t.Fatalf("round-trip lost fields: %+v / %+v", rt.Decision, rt.Details)
	}
}

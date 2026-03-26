package claude_test

import (
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

package copilot_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks/copilot"
)

func TestStopResponses(t *testing.T) {
	r := copilot.LetStop()
	if r.Proceed == nil || !*r.Proceed {
		t.Fatal("LetStop should set Proceed=true")
	}
	if r.Decision != "" {
		t.Fatal("LetStop should not set Decision")
	}

	b := copilot.HaltAndContinue("keep working")
	if b.Decision != "block" {
		t.Fatalf("HaltAndContinue: Decision=%q, want block", b.Decision)
	}
	if b.Reason != "keep working" {
		t.Fatalf("HaltAndContinue: Reason=%q", b.Reason)
	}
}

func TestPreToolUseResponses(t *testing.T) {
	a := copilot.ApproveToolUse()
	if a.Details == nil || a.Details.Decision != "allow" {
		t.Fatal("ApproveToolUse should set decision=allow")
	}

	d := copilot.DenyToolUse("dangerous")
	if d.Details == nil || d.Details.Decision != "deny" {
		t.Fatal("DenyToolUse should set decision=deny")
	}
	if d.Details.DecisionReason != "dangerous" {
		t.Fatalf("DenyToolUse: DecisionReason=%q", d.Details.DecisionReason)
	}

	ask := copilot.AskUserAboutTool("confirm?")
	if ask.Details == nil || ask.Details.Decision != "ask" {
		t.Fatal("AskUserAboutTool should set decision=ask")
	}

	note := copilot.ApproveToolUseWithNote("ok with note")
	if note.Details == nil || note.Details.DecisionReason != "ok with note" {
		t.Fatal("ApproveToolUseWithNote should carry the note")
	}
}

func TestUserPromptSubmitResponses(t *testing.T) {
	a := copilot.ApprovePrompt()
	if a.Proceed == nil || !*a.Proceed {
		t.Fatal("ApprovePrompt should set Proceed=true")
	}

	r := copilot.RejectPrompt("blocked")
	if r.Decision != "block" {
		t.Fatalf("RejectPrompt: Decision=%q, want block", r.Decision)
	}

	e := copilot.AppendToPrompt("extra context")
	if e.Details == nil || e.Details.ExtraContext != "extra context" {
		t.Fatal("AppendToPrompt should set ExtraContext")
	}
}

func TestPostToolUseResponses(t *testing.T) {
	a := copilot.AcknowledgeToolUse()
	if a.Decision != "" || a.Details != nil {
		t.Fatal("AcknowledgeToolUse should be empty")
	}

	c := copilot.AddToolContext("note")
	if c.Details == nil || c.Details.ExtraContext != "note" {
		t.Fatal("AddToolContext should set ExtraContext")
	}

	r := copilot.RejectToolResult("bad edit")
	if r.Decision != "block" {
		t.Fatalf("RejectToolResult: Decision=%q", r.Decision)
	}
}

func TestSessionStartResponses(t *testing.T) {
	if copilot.AcknowledgeSession().Details != nil {
		t.Fatal("AcknowledgeSession should be empty")
	}
	c := copilot.InjectSessionContext("hello")
	if c.Details == nil || c.Details.ExtraContext != "hello" {
		t.Fatal("InjectSessionContext should set ExtraContext")
	}
}

func TestSubagentStopResponses(t *testing.T) {
	if copilot.LetSubagentStop().Decision != "" {
		t.Fatal("LetSubagentStop should not set Decision")
	}
	b := copilot.KeepSubagentRunning("not done")
	if b.Decision != "block" || b.Reason != "not done" {
		t.Fatalf("KeepSubagentRunning: %+v", b)
	}
}

// TestEventBaseUnmarshalsCamelCase verifies that Copilot's camelCase keys
// (sessionId, hookEventName) parse correctly — this is the main divergence
// from the Claude Code protocol.
func TestEventBaseUnmarshalsCamelCase(t *testing.T) {
	raw := `{
		"timestamp": "2026-04-29T10:00:00Z",
		"cwd": "/repo",
		"sessionId": "sess-123",
		"hookEventName": "PreToolUse",
		"transcript_path": "/tmp/t.jsonl",
		"tool_name": "shell",
		"tool_input": {"command": "ls"},
		"tool_use_id": "tu-1"
	}`
	var ev copilot.PreToolUseEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.SessionID != "sess-123" {
		t.Fatalf("SessionID=%q", ev.SessionID)
	}
	if ev.EventName != "PreToolUse" {
		t.Fatalf("EventName=%q", ev.EventName)
	}
	if ev.WorkDir != "/repo" {
		t.Fatalf("WorkDir=%q", ev.WorkDir)
	}
	if ev.ToolName != "shell" {
		t.Fatalf("ToolName=%q", ev.ToolName)
	}
	if !strings.Contains(string(ev.ToolInput), "command") {
		t.Fatalf("ToolInput not preserved: %s", ev.ToolInput)
	}
}

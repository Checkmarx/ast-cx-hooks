package copilot_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Checkmarx/ast-cx-hooks/copilot"
)

func TestStopResponses(t *testing.T) {
	r := copilot.LetStop()
	if r.Proceed == nil || !*r.Proceed {
		t.Fatal("LetStop should set Proceed=true")
	}
	if r.Details != nil {
		t.Fatal("LetStop should not set hookSpecificOutput")
	}

	b := copilot.HaltAndContinue("keep working")
	if b.Details == nil || b.Details.Decision != "block" {
		t.Fatalf("HaltAndContinue should nest decision=block under hookSpecificOutput, got %+v", b.Details)
	}
	if b.Details.Reason != "keep working" {
		t.Fatalf("HaltAndContinue: Reason=%q", b.Details.Reason)
	}
	if b.Details.EventName != "Stop" {
		t.Fatalf("HaltAndContinue: EventName=%q, want Stop", b.Details.EventName)
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

	allowCtx := copilot.ApproveToolUseWithContext("run the remediation skill")
	if allowCtx.Details == nil || allowCtx.Details.Decision != "allow" {
		t.Fatal("ApproveToolUseWithContext should set decision=allow")
	}
	if allowCtx.Details.ExtraContext != "run the remediation skill" {
		t.Fatalf("ApproveToolUseWithContext: ExtraContext=%q", allowCtx.Details.ExtraContext)
	}
}

// TestDenyToolUseWithContextJSON verifies the deny-with-context builder emits
// BOTH the deny decision (+ its reason) AND the additionalContext field, since
// Copilot's PreToolUse permission payload carries additionalContext.
func TestDenyToolUseWithContextJSON(t *testing.T) {
	r := copilot.DenyToolUseWithContext("blocked: secret detected", "run /cx-remediate on the leaked secret")

	if r.Details == nil || r.Details.Decision != "deny" {
		t.Fatalf("DenyToolUseWithContext should set decision=deny, got %+v", r.Details)
	}
	if r.Details.DecisionReason != "blocked: secret detected" {
		t.Fatalf("DecisionReason=%q", r.Details.DecisionReason)
	}
	if r.Details.ExtraContext != "run /cx-remediate on the leaked secret" {
		t.Fatalf("ExtraContext=%q", r.Details.ExtraContext)
	}

	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	js := string(out)
	if !strings.Contains(js, `"permissionDecision":"deny"`) {
		t.Fatalf("JSON missing deny decision: %s", js)
	}
	if !strings.Contains(js, `"permissionDecisionReason":"blocked: secret detected"`) {
		t.Fatalf("JSON missing deny reason: %s", js)
	}
	if !strings.Contains(js, `"additionalContext":"run /cx-remediate on the leaked secret"`) {
		t.Fatalf("JSON missing additionalContext: %s", js)
	}
}

// TestRejectToolResultWithContextJSON verifies the post-write reject-with-context
// builder emits BOTH the block decision (+ reason) AND additionalContext, since
// Copilot's PostToolUse payload carries additionalContext under hookSpecificOutput.
func TestRejectToolResultWithContextJSON(t *testing.T) {
	r := copilot.RejectToolResultWithContext("rejected: SQL injection introduced", "run /cx-remediate on finding CX-123")

	if r.Decision != "block" || r.Reason != "rejected: SQL injection introduced" {
		t.Fatalf("RejectToolResultWithContext block/reason: %+v", r)
	}
	if r.Details == nil || r.Details.ExtraContext != "run /cx-remediate on finding CX-123" {
		t.Fatalf("RejectToolResultWithContext should carry additionalContext: %+v", r.Details)
	}

	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	js := string(out)
	if !strings.Contains(js, `"decision":"block"`) {
		t.Fatalf("JSON missing block decision: %s", js)
	}
	if !strings.Contains(js, `"reason":"rejected: SQL injection introduced"`) {
		t.Fatalf("JSON missing reason: %s", js)
	}
	if !strings.Contains(js, `"additionalContext":"run /cx-remediate on finding CX-123"`) {
		t.Fatalf("JSON missing additionalContext: %s", js)
	}
}

func TestUserPromptSubmitResponses(t *testing.T) {
	a := copilot.ApprovePrompt()
	if a.Proceed == nil || !*a.Proceed {
		t.Fatal("ApprovePrompt should set Proceed=true")
	}

	// VS Code Copilot UserPromptSubmit is common-output-only: rejection uses continue=false.
	r := copilot.RejectPrompt("blocked")
	if r.Proceed == nil || *r.Proceed {
		t.Fatal("RejectPrompt should set continue=false")
	}
	if r.HaltReason != "blocked" {
		t.Fatalf("RejectPrompt: HaltReason=%q, want blocked", r.HaltReason)
	}

	// AppendToPrompt is a documented no-op on this platform (no additionalContext support).
	e := copilot.AppendToPrompt("extra context")
	if e.Proceed == nil || !*e.Proceed {
		t.Fatal("AppendToPrompt should be a no-op that lets the prompt proceed")
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

func TestSubagentStartResponses(t *testing.T) {
	if copilot.AcknowledgeSubagentStart().Details != nil {
		t.Fatal("AcknowledgeSubagentStart should be empty")
	}
	c := copilot.InjectSubagentContext("scope note")
	if c.Details == nil || c.Details.ExtraContext != "scope note" {
		t.Fatalf("InjectSubagentContext should set additionalContext: %+v", c.Details)
	}
	if c.Details.EventName != "SubagentStart" {
		t.Fatalf("EventName=%q", c.Details.EventName)
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

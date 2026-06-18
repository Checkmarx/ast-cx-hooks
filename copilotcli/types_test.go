package copilotcli

import (
	"encoding/json"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// These tests pin the live GitHub Copilot CLI preToolUse payload shape (captured
// from v1.0.62): camelCase keys, a NUMERIC timestamp, and toolArgs as a
// JSON-encoded string. Before the fix this decode failed outright on timestamp,
// and even when forced left ToolName/ToolInput empty so create/edit went unscanned.

func TestPreToolUse_CopilotCLIEdit(t *testing.T) {
	// toolArgs is a JSON-encoded STRING (note the escaped quotes) with path/old_str/new_str.
	raw := `{"sessionId":"s-1","timestamp":1781522130891,"cwd":"/repo",` +
		`"toolName":"edit","toolArgs":"{\"path\":\"/repo/Login.java\",\"old_str\":\"a\",\"new_str\":\"b\"}"}`

	var ev PreToolUseEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("decode failed (numeric timestamp regression?): %v", err)
	}
	if ev.ToolName != "edit" {
		t.Fatalf("ToolName = %q, want %q", ev.ToolName, "edit")
	}
	if ev.SessionID != "s-1" {
		t.Fatalf("SessionID = %q, want %q (sessionId not mapped)", ev.SessionID, "s-1")
	}
	if !hookcore.CopilotCLITools.IsWrite(ev.ToolName) {
		t.Fatal("IsWrite(edit) = false, want true")
	}
	if got := hookcore.CopilotCLITools.FilePath(ev.ToolInput); got != "/repo/Login.java" {
		t.Fatalf("FilePath = %q, want %q (toolArgs not unwrapped or 'path' key missing)", got, "/repo/Login.java")
	}
	ch := hookcore.CopilotCLITools.Changes(ev.ToolName, ev.ToolInput)
	if len(ch) != 1 || ch[0].Before != "a" || ch[0].After != "b" {
		t.Fatalf("Changes = %+v, want one diff before=a after=b (old_str/new_str)", ch)
	}
}

func TestPreToolUse_CopilotCLICreateAndShell(t *testing.T) {
	create := `{"sessionId":"s","timestamp":1,"cwd":"/r","toolName":"create",` +
		`"toolArgs":"{\"path\":\"/r/New.go\",\"file_text\":\"package main\"}"}`
	var ce PreToolUseEvent
	if err := json.Unmarshal([]byte(create), &ce); err != nil {
		t.Fatalf("create decode: %v", err)
	}
	if got := hookcore.CopilotCLITools.FilePath(ce.ToolInput); got != "/r/New.go" {
		t.Fatalf("create FilePath = %q, want /r/New.go", got)
	}
	if ch := hookcore.CopilotCLITools.Changes("create", ce.ToolInput); len(ch) != 1 || ch[0].After != "package main" {
		t.Fatalf("create Changes = %+v, want After=package main (file_text)", ch)
	}

	shell := `{"sessionId":"s","timestamp":2,"cwd":"/r","toolName":"powershell",` +
		`"toolArgs":"{\"command\":\"echo hi\"}"}`
	var se PreToolUseEvent
	if err := json.Unmarshal([]byte(shell), &se); err != nil {
		t.Fatalf("shell decode: %v", err)
	}
	kind, cmd := hookcore.CopilotCLITools.Kind(se.ToolName, se.ToolInput)
	if kind != hookcore.ToolKindShell || cmd != "echo hi" {
		t.Fatalf("Kind = (%v,%q), want (shell,'echo hi')", kind, cmd)
	}
}

// TestPreToolUse_VSCodeFormatStillWorks guards backward compatibility with the
// snake_case / object tool_input / string timestamp shape.
func TestPreToolUse_VSCodeFormatStillWorks(t *testing.T) {
	raw := `{"session_id":"s","timestamp":"2026-06-16T00:00:00Z","cwd":"/r",` +
		`"tool_name":"edit","tool_input":{"file_path":"/r/X.go","old_string":"x","new_string":"y"}}`
	var ev PreToolUseEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("VS Code decode: %v", err)
	}
	if ev.ToolName != "edit" || ev.SessionID != "s" {
		t.Fatalf("got ToolName=%q SessionID=%q", ev.ToolName, ev.SessionID)
	}
	if got := hookcore.CopilotCLITools.FilePath(ev.ToolInput); got != "/r/X.go" {
		t.Fatalf("FilePath = %q, want /r/X.go (file_path fallback)", got)
	}
}

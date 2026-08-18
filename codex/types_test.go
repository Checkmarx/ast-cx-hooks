package codex

import (
	"encoding/json"
	"testing"

	"github.com/Checkmarx/ast-cx-hooks/internal/hookcore"
)

// These tests decode hand-built sample payloads that follow the field names
// documented at https://learn.chatgpt.com/docs/hooks. They are NOT captured
// from a live Codex CLI process — see codex/doc.go for the list of assumptions
// that still need verification against a real payload.

func TestStopEventDecode(t *testing.T) {
	raw := `{"session_id":"s-1","cwd":"/repo","hook_event_name":"Stop","model":"gpt-5-codex","turn_id":"t-1"}`
	var ev StopEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.SessionID != "s-1" || ev.WorkDir != "/repo" || ev.Model != "gpt-5-codex" || ev.TurnID != "t-1" {
		t.Fatalf("got %+v", ev)
	}
}

func TestPreToolUseEventDecode(t *testing.T) {
	raw := `{"session_id":"s-2","cwd":"/repo","tool_name":"Bash","tool_use_id":"tu-1",` +
		`"tool_input":{"command":"rm -rf /"}}`
	var ev PreToolUseEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.ToolName != "Bash" || ev.ToolUseID != "tu-1" {
		t.Fatalf("got %+v", ev)
	}
	kind, cmd := hookcore.CodexTools.Kind(ev.ToolName, ev.ToolInput)
	if kind != hookcore.ToolKindShell || cmd != "rm -rf /" {
		t.Fatalf("Kind = (%v,%q), want (shell,'rm -rf /')", kind, cmd)
	}
}

func TestPreToolUseEventDecode_ApplyPatch(t *testing.T) {
	raw := `{"session_id":"s-3","cwd":"/repo","tool_name":"apply_patch",` +
		`"tool_input":{"input":"*** Begin Patch\n*** Update File: a.go\n*** End Patch"}}`
	var ev PreToolUseEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hookcore.CodexTools.IsWrite(ev.ToolName) {
		t.Fatal("IsWrite(apply_patch) = false, want true")
	}
	ch := hookcore.CodexTools.Changes(ev.ToolName, ev.ToolInput)
	if len(ch) != 1 || ch[0].Before != "" || ch[0].After == "" {
		t.Fatalf("Changes = %+v, want one diff with non-empty After (raw patch text)", ch)
	}
}

func TestPostToolUseEventDecode(t *testing.T) {
	raw := `{"session_id":"s-4","cwd":"/repo","tool_name":"apply_patch",` +
		`"tool_input":{"input":"patch"},"tool_response":{"ok":true},"tool_use_id":"tu-2"}`
	var ev PostToolUseEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.ToolName != "apply_patch" || ev.ToolUseID != "tu-2" {
		t.Fatalf("got %+v", ev)
	}
}

func TestUserPromptSubmitEventDecode(t *testing.T) {
	raw := `{"session_id":"s-5","hook_event_name":"UserPromptSubmit","prompt":"hello"}`
	var ev UserPromptSubmitEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.Prompt != "hello" {
		t.Fatalf("got %+v", ev)
	}
}

func TestSubagentStopEventDecode(t *testing.T) {
	raw := `{"session_id":"s-6","agent_id":"a-1","agent_type":"reviewer","agent_transcript_path":"/tmp/a.jsonl"}`
	var ev SubagentStopEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.AgentID != "a-1" || ev.AgentType != "reviewer" {
		t.Fatalf("got %+v", ev)
	}
}

package claude_test

import (
	"encoding/json"
	"testing"

	"github.com/Checkmarx/ast-cx-hooks/claude"
)

// TestPostToolUseEventToolResponse verifies the modeled tool_response key captures
// a non-trivial structured value (not just the empty {} the existing E2E tests use).
//
// NOTE: the Claude reference docs show some inconsistency between `tool_response`
// and `tool_result` for this field; the local model uses `tool_response`. This test
// pins the current contract — confirm against a captured live PostToolUse payload
// before changing the tag.
func TestPostToolUseEventToolResponse(t *testing.T) {
	raw := `{
		"session_id":"s","cwd":"/r","hook_event_name":"PostToolUse",
		"tool_name":"Bash","tool_input":{"command":"ls"},
		"tool_response":{"output":"file1\nfile2","exitCode":0},
		"tool_use_id":"tu-1"
	}`
	var ev claude.PostToolUseEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ToolName != "Bash" || ev.ToolUseID != "tu-1" {
		t.Fatalf("base fields: name=%q id=%q", ev.ToolName, ev.ToolUseID)
	}
	var resp map[string]any
	if err := json.Unmarshal(ev.ToolResponse, &resp); err != nil {
		t.Fatalf("tool_response not preserved as raw JSON: %v", err)
	}
	if resp["output"] != "file1\nfile2" {
		t.Fatalf("tool_response content not captured: %+v", resp)
	}
}

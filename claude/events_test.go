package claude

import (
	"encoding/json"
	"testing"
)

// --- Input unmarshalling ---

func TestUnmarshalSubagentStopEvent(t *testing.T) {
	payload := `{
		"session_id": "s1",
		"transcript_path": "/t/main.jsonl",
		"cwd": "/w",
		"permission_mode": "default",
		"hook_event_name": "SubagentStop",
		"agent_id": "a1",
		"agent_type": "researcher",
		"agent_transcript_path": "/t/sub.jsonl",
		"stop_hook_active": true
	}`
	var e SubagentStopEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.AgentID != "a1" || e.AgentType != "researcher" {
		t.Errorf("agent fields: %+v", e)
	}
	if e.AgentTranscriptPath != "/t/sub.jsonl" {
		t.Errorf("agent_transcript_path = %q", e.AgentTranscriptPath)
	}
	if !e.HookActive {
		t.Errorf("stop_hook_active not parsed")
	}
	if e.SessionID != "s1" {
		t.Errorf("EventBase not embedded: %q", e.SessionID)
	}
}

func TestUnmarshalSubagentStartEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "SubagentStart",
		"agent_id": "a2",
		"agent_type": "coder"
	}`
	var e SubagentStartEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.AgentID != "a2" || e.AgentType != "coder" {
		t.Errorf("agent fields: %+v", e)
	}
}

func TestUnmarshalPostToolUseFailureEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "PostToolUseFailure",
		"tool_name": "Bash",
		"tool_input": {"command": "ls"},
		"error": "exit 1",
		"tool_use_id": "tu1",
		"agent_id": "a3",
		"agent_type": "main"
	}`
	var e PostToolUseFailureEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.ToolName != "Bash" || e.Error != "exit 1" || e.ToolUseID != "tu1" {
		t.Errorf("fields: %+v", e)
	}
	if e.AgentID != "a3" || e.AgentType != "main" {
		t.Errorf("agent fields: %+v", e)
	}
	if string(e.ToolInput) != `{"command": "ls"}` {
		t.Errorf("tool_input = %s", e.ToolInput)
	}
}

func TestUnmarshalPermissionRequestEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "PermissionRequest",
		"tool_name": "Write",
		"tool_input": {"path": "/x"},
		"permission_mode": "acceptEdits"
	}`
	var e PermissionRequestEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.ToolName != "Write" || e.PermissionMode != "acceptEdits" {
		t.Errorf("fields: %+v", e)
	}
	if string(e.ToolInput) != `{"path": "/x"}` {
		t.Errorf("tool_input = %s", e.ToolInput)
	}
}

func TestUnmarshalPostCompactEvent(t *testing.T) {
	payload := `{"hook_event_name": "PostCompact", "trigger": "auto"}`
	var e PostCompactEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Trigger != "auto" {
		t.Errorf("trigger = %q", e.Trigger)
	}
}

// --- Builder marshalling ---

// marshalMap marshals v and unmarshals into a generic map for key-level assertions.
func marshalMap(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	return m
}

func TestLetSubagentStop(t *testing.T) {
	m := marshalMap(t, LetSubagentStop())
	if m["continue"] != true {
		t.Errorf("continue = %v", m["continue"])
	}
}

func TestHaltSubagent(t *testing.T) {
	m := marshalMap(t, HaltSubagent("keep going"))
	if m["decision"] != "block" || m["reason"] != "keep going" {
		t.Errorf("got %v", m)
	}
}

func TestAcknowledgeSubagentStart(t *testing.T) {
	b, _ := json.Marshal(AcknowledgeSubagentStart())
	if string(b) != "{}" {
		t.Errorf("expected empty object, got %s", b)
	}
}

func TestInjectSubagentContext(t *testing.T) {
	m := marshalMap(t, InjectSubagentContext("ctx"))
	out, ok := m["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing hookSpecificOutput: %v", m)
	}
	if out["hookEventName"] != "SubagentStart" || out["additionalContext"] != "ctx" {
		t.Errorf("got %v", out)
	}
}

func TestAcknowledgeToolFailure(t *testing.T) {
	b, _ := json.Marshal(AcknowledgeToolFailure())
	if string(b) != "{}" {
		t.Errorf("expected empty object, got %s", b)
	}
}

func TestAnnotateToolFailure(t *testing.T) {
	m := marshalMap(t, AnnotateToolFailure("note"))
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["hookEventName"] != "PostToolUseFailure" || out["additionalContext"] != "note" {
		t.Errorf("got %v", out)
	}
}

func TestRejectAfterFailure(t *testing.T) {
	m := marshalMap(t, RejectAfterFailure("bad"))
	if m["decision"] != "block" || m["reason"] != "bad" {
		t.Errorf("got %v", m)
	}
}

func TestAllowPermission(t *testing.T) {
	m := marshalMap(t, AllowPermission(json.RawMessage(`{"path":"/y"}`)))
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["hookEventName"] != "PermissionRequest" {
		t.Errorf("hookEventName = %v", out["hookEventName"])
	}
	dec, ok := out["decision"].(map[string]interface{})
	if !ok {
		t.Fatalf("decision not nested object: %v", out)
	}
	if dec["behavior"] != "allow" {
		t.Errorf("behavior = %v", dec["behavior"])
	}
	upd, ok := dec["updatedInput"].(map[string]interface{})
	if !ok || upd["path"] != "/y" {
		t.Errorf("updatedInput = %v", dec["updatedInput"])
	}
}

func TestAllowPermissionNoInput(t *testing.T) {
	m := marshalMap(t, AllowPermission(nil))
	dec := m["hookSpecificOutput"].(map[string]interface{})["decision"].(map[string]interface{})
	if dec["behavior"] != "allow" {
		t.Errorf("behavior = %v", dec["behavior"])
	}
	if _, present := dec["updatedInput"]; present {
		t.Errorf("updatedInput should be omitted when nil")
	}
}

func TestDenyPermission(t *testing.T) {
	m := marshalMap(t, DenyPermission())
	dec := m["hookSpecificOutput"].(map[string]interface{})["decision"].(map[string]interface{})
	if dec["behavior"] != "deny" {
		t.Errorf("behavior = %v", dec["behavior"])
	}
}

func TestAcknowledgeCompact(t *testing.T) {
	m := marshalMap(t, AcknowledgeCompact())
	if m["continue"] != true {
		t.Errorf("continue = %v", m["continue"])
	}
}

// --- Extended existing events ---

func TestSetSessionTitle(t *testing.T) {
	m := marshalMap(t, SetSessionTitle("My Session"))
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["sessionTitle"] != "My Session" {
		t.Errorf("sessionTitle = %v", out["sessionTitle"])
	}
}

func TestSeedInitialMessage(t *testing.T) {
	m := marshalMap(t, SeedInitialMessage("hello"))
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["initialUserMessage"] != "hello" {
		t.Errorf("initialUserMessage = %v", out["initialUserMessage"])
	}
}

func TestWatchFiles(t *testing.T) {
	m := marshalMap(t, WatchFiles("/a", "/b"))
	out := m["hookSpecificOutput"].(map[string]interface{})
	paths, ok := out["watchPaths"].([]interface{})
	if !ok || len(paths) != 2 || paths[0] != "/a" || paths[1] != "/b" {
		t.Errorf("watchPaths = %v", out["watchPaths"])
	}
}

func TestReloadSkills(t *testing.T) {
	m := marshalMap(t, ReloadSkills())
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["reloadSkills"] != true {
		t.Errorf("reloadSkills = %v", out["reloadSkills"])
	}
}

func TestSetPromptSessionTitle(t *testing.T) {
	m := marshalMap(t, SetPromptSessionTitle("T"))
	if m["continue"] != true {
		t.Errorf("continue = %v", m["continue"])
	}
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["sessionTitle"] != "T" {
		t.Errorf("sessionTitle = %v", out["sessionTitle"])
	}
}

func TestRejectAndSuppressPrompt(t *testing.T) {
	m := marshalMap(t, RejectAndSuppressPrompt("no"))
	if m["decision"] != "block" || m["reason"] != "no" {
		t.Errorf("got %v", m)
	}
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["suppressOriginalPrompt"] != true {
		t.Errorf("suppressOriginalPrompt = %v", out["suppressOriginalPrompt"])
	}
}

func TestApproveToolUseWithContext(t *testing.T) {
	m := marshalMap(t, ApproveToolUseWithContext("more"))
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["permissionDecision"] != "allow" || out["additionalContext"] != "more" {
		t.Errorf("got %v", out)
	}
}

func TestDeferToolUse(t *testing.T) {
	m := marshalMap(t, DeferToolUse("later"))
	out := m["hookSpecificOutput"].(map[string]interface{})
	if out["permissionDecision"] != "defer" || out["permissionDecisionReason"] != "later" {
		t.Errorf("got %v", out)
	}
}

package cursor

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- Unmarshal tests: verify representative payloads parse key fields ---

func TestUnmarshalToolPreEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "preToolUse",
		"tool_name": "edit_file",
		"tool_input": {"path": "main.go"},
		"tool_use_id": "tu_123",
		"cwd": "/repo",
		"model": "claude",
		"agent_message": "editing file"
	}`
	var e ToolPreEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.ToolName != "edit_file" || e.ToolUseID != "tu_123" || e.WorkDir != "/repo" ||
		e.ToolModel != "claude" || e.AgentMessage != "editing file" {
		t.Errorf("unexpected fields: %+v", e)
	}
	if string(e.ToolInput) == "" {
		t.Errorf("tool_input not captured")
	}
}

func TestUnmarshalToolPostEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "postToolUse",
		"tool_name": "read_file",
		"tool_input": {"path": "a.txt"},
		"tool_output": {"content": "hi"},
		"tool_use_id": "tu_9",
		"cwd": "/repo",
		"duration": 42,
		"model": "claude"
	}`
	var e ToolPostEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.ToolName != "read_file" || e.ToolUseID != "tu_9" || e.Duration != 42 || e.ToolModel != "claude" {
		t.Errorf("unexpected fields: %+v", e)
	}
	if string(e.ToolOutput) == "" {
		t.Errorf("tool_output not captured")
	}
}

func TestUnmarshalToolFailureEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "postToolUseFailure",
		"tool_name": "shell",
		"tool_input": {"cmd": "ls"},
		"tool_use_id": "tu_1",
		"cwd": "/repo",
		"error_message": "boom",
		"failure_type": "timeout",
		"duration": 100,
		"is_interrupt": true
	}`
	var e ToolFailureEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.ErrorMessage != "boom" || e.FailureType != "timeout" || e.Duration != 100 || !e.IsInterrupt {
		t.Errorf("unexpected fields: %+v", e)
	}
}

func TestUnmarshalReadFilePreEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "beforeReadFile",
		"file_path": "secret.env",
		"content": "KEY=1",
		"attachments": [{"type": "file", "file_path": "ref.txt"}]
	}`
	var e ReadFilePreEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.FilePath != "secret.env" || e.Content != "KEY=1" {
		t.Errorf("unexpected fields: %+v", e)
	}
	if len(e.Attachments) != 1 || e.Attachments[0].FilePath != "ref.txt" {
		t.Errorf("attachments not parsed: %+v", e.Attachments)
	}
}

func TestUnmarshalSubagentStartEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "subagentStart",
		"subagent_id": "sa_1",
		"subagent_type": "worker",
		"task": "do thing",
		"parent_conversation_id": "conv_1",
		"tool_call_id": "tc_1",
		"subagent_model": "haiku",
		"is_parallel_worker": true,
		"git_branch": "feature/x"
	}`
	var e SubagentStartEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.SubagentID != "sa_1" || e.SubagentType != "worker" || e.Task != "do thing" ||
		e.ParentConversationID != "conv_1" || e.ToolCallID != "tc_1" ||
		e.SubagentModel != "haiku" || !e.IsParallelWorker || e.GitBranch != "feature/x" {
		t.Errorf("unexpected fields: %+v", e)
	}
}

func TestUnmarshalSubagentStopEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "subagentStop",
		"subagent_type": "worker",
		"status": "completed",
		"task": "do thing",
		"description": "did thing",
		"summary": "done",
		"duration_ms": 1500,
		"message_count": 4,
		"tool_call_count": 7,
		"loop_count": 2,
		"modified_files": ["a.go", "b.go"],
		"agent_transcript_path": "/tmp/t.json"
	}`
	var e SubagentStopEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.SubagentType != "worker" || e.Status != "completed" || e.DurationMS != 1500 ||
		e.MessageCount != 4 || e.ToolCallCount != 7 || e.LoopCount != 2 ||
		e.AgentTranscriptPath != "/tmp/t.json" {
		t.Errorf("unexpected fields: %+v", e)
	}
	if len(e.ModifiedFiles) != 2 {
		t.Errorf("modified_files not parsed: %+v", e.ModifiedFiles)
	}
}

func TestUnmarshalAgentResponseEvent(t *testing.T) {
	payload := `{"hook_event_name": "afterAgentResponse", "text": "hello"}`
	var e AgentResponseEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Text != "hello" {
		t.Errorf("unexpected text: %q", e.Text)
	}
}

func TestUnmarshalAgentThoughtEvent(t *testing.T) {
	payload := `{"hook_event_name": "afterAgentThought", "text": "thinking", "duration_ms": 250}`
	var e AgentThoughtEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Text != "thinking" || e.DurationMS != 250 {
		t.Errorf("unexpected fields: %+v", e)
	}
}

func TestUnmarshalShellSandboxField(t *testing.T) {
	pre := `{"hook_event_name": "beforeShellExecution", "command": "ls", "cwd": "/r", "timeout": 30, "sandbox": true}`
	var preEv ShellPreEvent
	if err := json.Unmarshal([]byte(pre), &preEv); err != nil {
		t.Fatalf("unmarshal pre: %v", err)
	}
	if !preEv.Sandbox {
		t.Errorf("ShellPreEvent.Sandbox not parsed")
	}

	post := `{"hook_event_name": "afterShellExecution", "command": "ls", "output": "x", "duration": 5, "sandbox": true}`
	var postEv ShellPostEvent
	if err := json.Unmarshal([]byte(post), &postEv); err != nil {
		t.Fatalf("unmarshal post: %v", err)
	}
	if !postEv.Sandbox {
		t.Errorf("ShellPostEvent.Sandbox not parsed")
	}
}

// --- Marshal tests: verify result builders emit correct snake_case keys ---

func marshalToString(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestMarshalResultBuilders(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		mustHave []string
	}{
		{"PermitTool", PermitTool(), []string{`"permission":"allow"`}},
		{"ForbidTool", ForbidTool("u", "a"), []string{`"permission":"deny"`, `"user_message":"u"`, `"agent_message":"a"`}},
		{"RewriteToolInput", RewriteToolInput(json.RawMessage(`{"path":"x"}`)), []string{`"updated_input":{"path":"x"}`}},
		{"RewriteToolOutput", RewriteToolOutput(json.RawMessage(`{"ok":true}`)), []string{`"updated_mcp_tool_output":{"ok":true}`}},
		{"AddContext", AddContext("extra"), []string{`"additional_context":"extra"`}},
		{"PermitRead", PermitRead(), []string{`"permission":"allow"`}},
		{"ForbidRead", ForbidRead("nope"), []string{`"permission":"deny"`, `"user_message":"nope"`}},
		{"PermitSubagent", PermitSubagent(), []string{`"permission":"allow"`}},
		{"ForbidSubagent", ForbidSubagent("no"), []string{`"permission":"deny"`, `"user_message":"no"`}},
		{"SendSubagentFollowup", SendSubagentFollowup("again"), []string{`"followup_message":"again"`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := marshalToString(t, tc.value)
			for _, want := range tc.mustHave {
				if !strings.Contains(out, want) {
					t.Errorf("expected %s to contain %s, got %s", tc.name, want, out)
				}
			}
		})
	}
}

func TestMarshalObservationalResultsAreEmpty(t *testing.T) {
	for _, v := range []interface{}{ToolFailureResult{}, AgentResponseResult{}, AgentThoughtResult{}} {
		if out := marshalToString(t, v); out != "{}" {
			t.Errorf("expected empty object, got %s", out)
		}
	}
}

func TestLetSubagentStopIsEmpty(t *testing.T) {
	if out := marshalToString(t, LetSubagentStop()); out != "{}" {
		t.Errorf("expected empty object, got %s", out)
	}
}

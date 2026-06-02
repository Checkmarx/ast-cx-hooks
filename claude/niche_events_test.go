package claude

import (
	"encoding/json"
	"testing"
)

// mustMarshal marshals v and fails the test on error, returning the JSON string.
func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

// --- Setup ---

func TestSetupEventUnmarshal(t *testing.T) {
	var ev SetupEvent
	if err := json.Unmarshal([]byte(`{"hook_event_name":"Setup","trigger":"init"}`), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Trigger != "init" {
		t.Errorf("Trigger = %q, want init", ev.Trigger)
	}
}

func TestInjectSetupContextMarshal(t *testing.T) {
	got := mustMarshal(t, InjectSetupContext("hello"))
	want := `{"hookSpecificOutput":{"hookEventName":"Setup","additionalContext":"hello"}}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if mustMarshal(t, AcknowledgeSetup()) != `{}` {
		t.Errorf("AcknowledgeSetup should marshal to {}")
	}
}

// --- InstructionsLoaded ---

func TestInstructionsLoadedEventUnmarshal(t *testing.T) {
	var ev InstructionsLoadedEvent
	payload := `{"hook_event_name":"InstructionsLoaded","file_path":"/CLAUDE.md","memory_type":"Project","load_reason":"startup","globs":["**/*.go"],"trigger_file_path":"/t","parent_file_path":"/p"}`
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.FilePath != "/CLAUDE.md" || ev.MemoryType != "Project" || ev.LoadReason != "startup" {
		t.Errorf("unexpected fields: %+v", ev)
	}
	if len(ev.Globs) != 1 || ev.Globs[0] != "**/*.go" {
		t.Errorf("Globs = %v", ev.Globs)
	}
	if ev.TriggerFilePath != "/t" || ev.ParentFilePath != "/p" {
		t.Errorf("trigger/parent paths = %q/%q", ev.TriggerFilePath, ev.ParentFilePath)
	}
	if mustMarshal(t, AcknowledgeInstructions()) != `{}` {
		t.Errorf("AcknowledgeInstructions should marshal to {}")
	}
}

// --- PermissionDenied ---

func TestPermissionDeniedEventUnmarshal(t *testing.T) {
	var ev PermissionDeniedEvent
	payload := `{"hook_event_name":"PermissionDenied","tool_name":"Bash","tool_input":{"cmd":"ls"},"permission_mode":"default"}`
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ToolName != "Bash" || ev.PermissionMode != "default" {
		t.Errorf("unexpected fields: %+v", ev)
	}
	if string(ev.ToolInput) != `{"cmd":"ls"}` {
		t.Errorf("ToolInput = %s", ev.ToolInput)
	}
}

func TestPermissionDeniedBuilders(t *testing.T) {
	got := mustMarshal(t, RequestRetry())
	want := `{"hookSpecificOutput":{"hookEventName":"PermissionDenied","retry":true}}`
	if got != want {
		t.Errorf("RequestRetry got %s, want %s", got, want)
	}
	got = mustMarshal(t, AcceptDenial())
	want = `{"hookSpecificOutput":{"hookEventName":"PermissionDenied"}}`
	if got != want {
		t.Errorf("AcceptDenial got %s, want %s", got, want)
	}
}

// --- ConfigChange ---

func TestConfigChangeEventUnmarshal(t *testing.T) {
	var ev ConfigChangeEvent
	payload := `{"hook_event_name":"ConfigChange","config_source":"project_settings","config_path":"/.claude/settings.json"}`
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ConfigSource != "project_settings" || ev.ConfigPath != "/.claude/settings.json" {
		t.Errorf("unexpected fields: %+v", ev)
	}
}

func TestConfigChangeBuilders(t *testing.T) {
	got := mustMarshal(t, BlockConfigChange("nope"))
	want := `{"decision":"block","reason":"nope"}`
	if got != want {
		t.Errorf("BlockConfigChange got %s, want %s", got, want)
	}
	got = mustMarshal(t, AllowConfigChange())
	want = `{"continue":true}`
	if got != want {
		t.Errorf("AllowConfigChange got %s, want %s", got, want)
	}
}

// --- CwdChanged ---

func TestCwdChangedEventUnmarshal(t *testing.T) {
	var ev CwdChangedEvent
	if err := json.Unmarshal([]byte(`{"hook_event_name":"CwdChanged","new_cwd":"/tmp/x"}`), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.NewCwd != "/tmp/x" {
		t.Errorf("NewCwd = %q", ev.NewCwd)
	}
	if mustMarshal(t, AcknowledgeCwdChange()) != `{}` {
		t.Errorf("AcknowledgeCwdChange should marshal to {}")
	}
}

// --- FileChanged ---

func TestFileChangedEventUnmarshal(t *testing.T) {
	var ev FileChangedEvent
	if err := json.Unmarshal([]byte(`{"hook_event_name":"FileChanged","file_path":"/a.go","change_type":"modified"}`), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.FilePath != "/a.go" || ev.ChangeType != "modified" {
		t.Errorf("unexpected fields: %+v", ev)
	}
	if mustMarshal(t, AcknowledgeFileChange()) != `{}` {
		t.Errorf("AcknowledgeFileChange should marshal to {}")
	}
}

// --- StopFailure ---

func TestStopFailureEventUnmarshal(t *testing.T) {
	var ev StopFailureEvent
	if err := json.Unmarshal([]byte(`{"hook_event_name":"StopFailure","error_type":"timeout","error_message":"boom"}`), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ErrorType != "timeout" || ev.ErrorMessage != "boom" {
		t.Errorf("unexpected fields: %+v", ev)
	}
	if mustMarshal(t, AcknowledgeStopFailure()) != `{}` {
		t.Errorf("AcknowledgeStopFailure should marshal to {}")
	}
}

// --- UserPromptExpansion ---

func TestUserPromptExpansionEventUnmarshal(t *testing.T) {
	var ev UserPromptExpansionEvent
	payload := `{"hook_event_name":"UserPromptExpansion","expansion_type":"slash_command","command_name":"ship","command_args":"--fast","command_source":"project","prompt":"do it"}`
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ExpansionType != "slash_command" || ev.CommandName != "ship" || ev.CommandArgs != "--fast" {
		t.Errorf("unexpected fields: %+v", ev)
	}
	if ev.CommandSource != "project" || ev.Prompt != "do it" {
		t.Errorf("unexpected fields: %+v", ev)
	}
}

func TestUserPromptExpansionBuilders(t *testing.T) {
	got := mustMarshal(t, AnnotateExpansion("note"))
	want := `{"continue":true,"hookSpecificOutput":{"hookEventName":"UserPromptExpansion","additionalContext":"note"}}`
	if got != want {
		t.Errorf("AnnotateExpansion got %s, want %s", got, want)
	}
	got = mustMarshal(t, BlockExpansion("bad"))
	want = `{"decision":"block","reason":"bad"}`
	if got != want {
		t.Errorf("BlockExpansion got %s, want %s", got, want)
	}
	if mustMarshal(t, AllowExpansion()) != `{"continue":true}` {
		t.Errorf("AllowExpansion unexpected")
	}
}

// --- WorktreeCreate ---

func TestWorktreeCreateEventUnmarshal(t *testing.T) {
	var ev WorktreeCreateEvent
	payload := `{"hook_event_name":"WorktreeCreate","worktree_path":"/wt","ref":"main","isolation_type":"branch"}`
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.WorktreePath != "/wt" || ev.Ref != "main" || ev.IsolationType != "branch" {
		t.Errorf("unexpected fields: %+v", ev)
	}
}

func TestWorktreeCreateBuilders(t *testing.T) {
	got := mustMarshal(t, SetWorktreePath("/new/wt"))
	want := `{"hookSpecificOutput":{"hookEventName":"WorktreeCreate","worktreePath":"/new/wt"}}`
	if got != want {
		t.Errorf("SetWorktreePath got %s, want %s", got, want)
	}
	if mustMarshal(t, AcknowledgeWorktreeCreate()) != `{}` {
		t.Errorf("AcknowledgeWorktreeCreate should marshal to {}")
	}
}

// --- WorktreeRemove ---

func TestWorktreeRemoveEventUnmarshal(t *testing.T) {
	var ev WorktreeRemoveEvent
	if err := json.Unmarshal([]byte(`{"hook_event_name":"WorktreeRemove","worktree_path":"/wt"}`), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.WorktreePath != "/wt" {
		t.Errorf("WorktreePath = %q", ev.WorktreePath)
	}
	if mustMarshal(t, AcknowledgeWorktreeRemove()) != `{}` {
		t.Errorf("AcknowledgeWorktreeRemove should marshal to {}")
	}
}

// --- Elicitation ---

func TestElicitationEventUnmarshal(t *testing.T) {
	var ev ElicitationEvent
	payload := `{"hook_event_name":"Elicitation","server_name":"srv","form_fields":[{"name":"x"}]}`
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ServerName != "srv" {
		t.Errorf("ServerName = %q", ev.ServerName)
	}
	if string(ev.FormFields) != `[{"name":"x"}]` {
		t.Errorf("FormFields = %s", ev.FormFields)
	}
}

func TestElicitationBuilders(t *testing.T) {
	got := mustMarshal(t, AcceptElicitation(json.RawMessage(`{"x":1}`)))
	want := `{"hookSpecificOutput":{"hookEventName":"Elicitation","action":"accept","content":{"x":1}}}`
	if got != want {
		t.Errorf("AcceptElicitation got %s, want %s", got, want)
	}
	got = mustMarshal(t, DeclineElicitation())
	want = `{"hookSpecificOutput":{"hookEventName":"Elicitation","action":"decline"}}`
	if got != want {
		t.Errorf("DeclineElicitation got %s, want %s", got, want)
	}
	got = mustMarshal(t, CancelElicitation())
	want = `{"hookSpecificOutput":{"hookEventName":"Elicitation","action":"cancel"}}`
	if got != want {
		t.Errorf("CancelElicitation got %s, want %s", got, want)
	}
}

// --- ElicitationResult ---

func TestElicitationResultEventUnmarshal(t *testing.T) {
	var ev ElicitationResultEvent
	payload := `{"hook_event_name":"ElicitationResult","server_name":"srv","form_values":{"x":"y"}}`
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ServerName != "srv" {
		t.Errorf("ServerName = %q", ev.ServerName)
	}
	if string(ev.FormValues) != `{"x":"y"}` {
		t.Errorf("FormValues = %s", ev.FormValues)
	}
}

func TestElicitationResultBuilders(t *testing.T) {
	got := mustMarshal(t, AcceptElicitationResult(json.RawMessage(`{"ok":true}`)))
	want := `{"hookSpecificOutput":{"hookEventName":"ElicitationResult","action":"accept","content":{"ok":true}}}`
	if got != want {
		t.Errorf("AcceptElicitationResult got %s, want %s", got, want)
	}
	got = mustMarshal(t, DeclineElicitationResult())
	want = `{"hookSpecificOutput":{"hookEventName":"ElicitationResult","action":"decline"}}`
	if got != want {
		t.Errorf("DeclineElicitationResult got %s, want %s", got, want)
	}
	got = mustMarshal(t, CancelElicitationResult())
	want = `{"hookSpecificOutput":{"hookEventName":"ElicitationResult","action":"cancel"}}`
	if got != want {
		t.Errorf("CancelElicitationResult got %s, want %s", got, want)
	}
}

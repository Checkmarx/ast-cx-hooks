package hookcore

import (
	"encoding/json"
	"testing"
)

// TestToolConventionKind locks the shell/MCP/builtin classification for every
// agent convention. A regression here silently misroutes tool-call gating across
// all platforms, so it is the highest-value invariant in this package.
func TestToolConventionKind(t *testing.T) {
	cases := []struct {
		name     string
		conv     ToolConvention
		tool     string
		input    string
		wantKind ToolKind
		wantCmd  string
	}{
		{"claude bash", ClaudeTools, "Bash", `{"command":"ls -la"}`, ToolKindShell, "ls -la"},
		{"claude mcp", ClaudeTools, "mcp__memory__store", `{}`, ToolKindMCP, ""},
		{"claude builtin", ClaudeTools, "Read", `{}`, ToolKindBuiltin, ""},
		{"droid execute", DroidTools, "Execute", `{"command":"go test"}`, ToolKindShell, "go test"},
		{"droid mcp double underscore", DroidTools, "mcp__gh__pr", `{}`, ToolKindMCP, ""},
		{"droid bash is not shell", DroidTools, "Bash", `{"command":"x"}`, ToolKindBuiltin, ""},
		{"gemini shell", GeminiTools, "run_shell_command", `{"command":"echo hi"}`, ToolKindShell, "echo hi"},
		{"gemini mcp single underscore", GeminiTools, "mcp_github_create", `{}`, ToolKindMCP, ""},
		{"gemini execute_bash is gone", GeminiTools, "execute_bash", `{"command":"x"}`, ToolKindBuiltin, ""},
		{"copilot terminal", CopilotTools, "runTerminalCommand", `{"command":"npm i"}`, ToolKindShell, "npm i"},
		{"copilot mcp single underscore", CopilotTools, "mcp_x_y", `{}`, ToolKindMCP, ""},
		{"copilot bash is not shell", CopilotTools, "Bash", `{"command":"x"}`, ToolKindBuiltin, ""},
		{"copilotcli bash lowercase", CopilotCLITools, "bash", `{"command":"rm -rf /"}`, ToolKindShell, "rm -rf /"},
		{"copilotcli powershell", CopilotCLITools, "powershell", `{"command":"gci"}`, ToolKindShell, "gci"},
		{"copilotcli no mcp prefix matching", CopilotCLITools, "mcp_x_y", `{}`, ToolKindBuiltin, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, cmd := tc.conv.Kind(tc.tool, json.RawMessage(tc.input))
			if kind != tc.wantKind {
				t.Fatalf("kind: got %v want %v", kind, tc.wantKind)
			}
			if cmd != tc.wantCmd {
				t.Fatalf("cmd: got %q want %q", cmd, tc.wantCmd)
			}
		})
	}
}

// TestToolConventionIsWrite locks each agent's file-write tool set — the gate that
// decides whether the post-file-write hook fires at all.
func TestToolConventionIsWrite(t *testing.T) {
	cases := []struct {
		conv  ToolConvention
		tool  string
		write bool
	}{
		{ClaudeTools, "Write", true}, {ClaudeTools, "Edit", true}, {ClaudeTools, "MultiEdit", true}, {ClaudeTools, "Create", false},
		{DroidTools, "Create", true}, {DroidTools, "Edit", true}, {DroidTools, "ApplyPatch", true}, {DroidTools, "Write", false},
		{GeminiTools, "write_file", true}, {GeminiTools, "replace", true}, {GeminiTools, "replace_in_file", false},
		{CopilotTools, "createFile", true}, {CopilotTools, "editFiles", true}, {CopilotTools, "Write", false},
		{CopilotCLITools, "create", true}, {CopilotCLITools, "edit", true}, {CopilotCLITools, "Create", false},
	}
	for _, tc := range cases {
		if got := tc.conv.IsWrite(tc.tool); got != tc.write {
			t.Errorf("IsWrite(%q): got %v want %v", tc.tool, got, tc.write)
		}
	}
}

// TestToolConventionFilePath covers the per-agent key (snake vs camel) and Gemini's
// file_path -> path fallback.
func TestToolConventionFilePath(t *testing.T) {
	if got := ClaudeTools.FilePath(json.RawMessage(`{"file_path":"/a.go"}`)); got != "/a.go" {
		t.Errorf("claude file_path: %q", got)
	}
	if got := CopilotTools.FilePath(json.RawMessage(`{"filePath":"/b.ts"}`)); got != "/b.ts" {
		t.Errorf("copilot filePath: %q", got)
	}
	if got := CopilotTools.FilePath(json.RawMessage(`{"file_path":"/snake.ts"}`)); got != "" {
		t.Errorf("copilot must not read snake_case file_path: %q", got)
	}
	if got := GeminiTools.FilePath(json.RawMessage(`{"path":"/c.py"}`)); got != "/c.py" {
		t.Errorf("gemini path fallback: %q", got)
	}
	if got := GeminiTools.FilePath(json.RawMessage(`{"file_path":"/d.py","path":"/ignored"}`)); got != "/d.py" {
		t.Errorf("gemini must prefer file_path over path: %q", got)
	}
}

// TestToolConventionChanges covers the diff extractors, including Droid's ApplyPatch
// fallback and Gemini's nil (no-diff) convention.
func TestToolConventionChanges(t *testing.T) {
	// standardDiff: Edit -> old/new
	if d := ClaudeTools.Changes("Edit", json.RawMessage(`{"old_string":"a","new_string":"b"}`)); d[0].Before != "a" || d[0].After != "b" {
		t.Errorf("claude edit diff: %+v", d)
	}
	// standardDiff: create-style -> content
	if d := ClaudeTools.Changes("Write", json.RawMessage(`{"content":"hi"}`)); d[0].Before != "" || d[0].After != "hi" {
		t.Errorf("claude write diff: %+v", d)
	}
	// droidDiff: ApplyPatch -> patch (else diff)
	if d := DroidTools.Changes("ApplyPatch", json.RawMessage(`{"patch":"@@ -1 +1 @@"}`)); d[0].After != "@@ -1 +1 @@" {
		t.Errorf("droid applypatch diff: %+v", d)
	}
	if d := DroidTools.Changes("ApplyPatch", json.RawMessage(`{"diff":"fallback"}`)); d[0].After != "fallback" {
		t.Errorf("droid applypatch diff fallback: %+v", d)
	}
	// cliDiff: lowercase edit -> old/new
	if d := CopilotCLITools.Changes("edit", json.RawMessage(`{"old_string":"x","new_string":"y"}`)); d[0].Before != "x" || d[0].After != "y" {
		t.Errorf("cli edit diff: %+v", d)
	}
	// gemini: no diff
	if d := GeminiTools.Changes("write_file", json.RawMessage(`{"content":"z"}`)); d != nil {
		t.Errorf("gemini should report no diffs, got %+v", d)
	}
}

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
		{"codex bash", CodexTools, "Bash", `{"command":"ls -la"}`, ToolKindShell, "ls -la"},
		{"codex mcp double underscore", CodexTools, "mcp__memory__store", `{}`, ToolKindMCP, ""},
		{"codex apply_patch is not shell", CodexTools, "apply_patch", `{"input":"x"}`, ToolKindBuiltin, ""},
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
		{CursorTools, "Write", true}, {CursorTools, "Edit", true}, {CursorTools, "StrReplace", true},
		{CursorTools, "EditNotebook", true}, {CursorTools, "Read", false},
		{CodexTools, "apply_patch", true}, {CodexTools, "Bash", false},
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
	if got := CursorTools.FilePath(json.RawMessage(`{"path":"/cli/Demo.java"}`)); got != "/cli/Demo.java" {
		t.Errorf("cursor path fallback: %q", got)
	}
	if got := CursorTools.FilePath(json.RawMessage(`{"file_path":"/ide/Demo.java","path":"/ignored"}`)); got != "/ide/Demo.java" {
		t.Errorf("cursor must prefer file_path over path: %q", got)
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
	// cliDiff: native Copilot CLI payload uses old_str/new_str (not old_string/new_string)
	if d := CopilotCLITools.Changes("edit", json.RawMessage(`{"old_str":"A","new_str":"B"}`)); d[0].Before != "A" || d[0].After != "B" {
		t.Errorf("cli edit diff old_str/new_str: %+v", d)
	}
	// cliDiff: old_str present but empty must NOT fall back to old_string —
	// empty string is a valid replacement target, not a missing key.
	if d := CopilotCLITools.Changes("edit", json.RawMessage(`{"old_str":"","new_str":"injected","old_string":"original","new_string":"ignored"}`)); d[0].Before != "" || d[0].After != "injected" {
		t.Errorf("cli edit diff: empty old_str must not fall back to old_string: %+v", d)
	}
	// cliDiff: create uses file_text; falls back to content when file_text is absent
	if d := CopilotCLITools.Changes("create", json.RawMessage(`{"file_text":"newfile"}`)); d[0].Before != "" || d[0].After != "newfile" {
		t.Errorf("cli create diff file_text: %+v", d)
	}
	if d := CopilotCLITools.Changes("create", json.RawMessage(`{"content":"fallback"}`)); d[0].Before != "" || d[0].After != "fallback" {
		t.Errorf("cli create diff content fallback: %+v", d)
	}
	// gemini: no diff
	if d := GeminiTools.Changes("write_file", json.RawMessage(`{"content":"z"}`)); d != nil {
		t.Errorf("gemini should report no diffs, got %+v", d)
	}
	// codexDiff: apply_patch with no file section yields no diffs.
	if d := CodexTools.Changes("apply_patch", json.RawMessage(`{"input":"*** Begin Patch\n*** End Patch"}`)); d != nil {
		t.Errorf("codex apply_patch diff with no file section should be nil: %+v", d)
	}
	// FilePath has no configured key for Codex (no per-file field exists), so it
	// falls back to codexPatchFilePath parsing the patch body itself; a patch
	// with no "*** Add/Update/Delete File:" header still yields "".
	if got := CodexTools.FilePath(json.RawMessage(`{"input":"*** Begin Patch"}`)); got != "" {
		t.Errorf("codex FilePath with no file header should be empty, got %q", got)
	}
	// command takes priority over input when both are present.
	if d := CodexTools.Changes("apply_patch", json.RawMessage(`{"command":"*** Begin Patch\n*** Add File: a.go\n+real\n*** End Patch","input":"*** Begin Patch\n*** Add File: a.go\n+stale\n*** End Patch"}`)); d[0].After != "real" {
		t.Errorf("codex apply_patch diff should prefer command over input: %+v", d)
	}
	// Add File: codexDiff reconstructs the full new file by stripping each
	// line's leading "+", confirmed against a live payload.
	if got := CodexTools.FilePath(json.RawMessage(`{"command":"*** Begin Patch\n*** Add File: heelo1.java\n+public class heelo1 {}\n*** End Patch\n"}`)); got != "heelo1.java" {
		t.Errorf("codex FilePath from patch header = %q, want %q", got, "heelo1.java")
	}
	if d := CodexTools.Changes("apply_patch", json.RawMessage(`{"command":"*** Begin Patch\n*** Add File: heelo1.go\n+package main\n+\n+func main() {}\n*** End Patch\n"}`)); len(d) != 1 || d[0].Before != "" || d[0].After != "package main\n\nfunc main() {}" {
		t.Errorf("codex Add File reconstruction: %+v", d)
	}
	// Add File with an absolute Windows path in the header, confirmed against a
	// live payload (C:\Users\...\heelo1.go).
	if got := CodexTools.FilePath(json.RawMessage(`{"command":"*** Begin Patch\n*** Add File: C:\\Users\\HiteshM\\heelo1.go\n+package main\n*** End Patch\n"}`)); got != `C:\Users\HiteshM\heelo1.go` {
		t.Errorf("codex FilePath absolute Windows path = %q", got)
	}
	// Update File: codexDiff resolves each "@@"-delimited hunk into a real
	// before/after pair (context lines shared, "-" removed, "+" added) instead
	// of surfacing raw patch syntax — confirmed against a live multi-hunk
	// payload (LoginController.java: new imports hunk + new method hunk).
	if got := CodexTools.FilePath(json.RawMessage(`{"command":"*** Begin Patch\n*** Update File: src/main/App.java\n@@\n-old\n+new\n*** End Patch\n"}`)); got != "src/main/App.java" {
		t.Errorf("codex FilePath from Update File header = %q, want %q", got, "src/main/App.java")
	}
	updatePatch := "*** Begin Patch\n" +
		"*** Update File: src/main/App.java\n" +
		"@@\n" +
		" import a;\n" +
		"+import java.sql.Connection;\n" +
		"@@\n" +
		"   void login() {\n" +
		"     ok();\n" +
		"   }\n" +
		"+\n" +
		"+  void debug() {\n" +
		"+    unsafe();\n" +
		"+  }\n" +
		"*** End Patch\n"
	updateInput, err := json.Marshal(map[string]string{"command": updatePatch})
	if err != nil {
		t.Fatalf("marshal updateInput: %v", err)
	}
	updateDiffs := CodexTools.Changes("apply_patch", updateInput)
	if len(updateDiffs) != 2 {
		t.Fatalf("codex Update File should yield one FileDiff per hunk, got %d: %+v", len(updateDiffs), updateDiffs)
	}
	if updateDiffs[0].Before != "import a;\n" || updateDiffs[0].After != "import a;\nimport java.sql.Connection;\n" {
		t.Errorf("codex Update File hunk 1: %+v", updateDiffs[0])
	}
	wantHunk2Before := "  void login() {\n    ok();\n  }\n"
	wantHunk2After := "  void login() {\n    ok();\n  }\n\n  void debug() {\n    unsafe();\n  }\n"
	if updateDiffs[1].Before != wantHunk2Before || updateDiffs[1].After != wantHunk2After {
		t.Errorf("codex Update File hunk 2:\n got before=%q after=%q\nwant before=%q after=%q",
			updateDiffs[1].Before, updateDiffs[1].After, wantHunk2Before, wantHunk2After)
	}
	// Multi-file patch: only the FIRST file section's path/diff is surfaced,
	// since FileDiff has no per-file grouping (mixing hunks from a second file
	// would apply against the wrong file's disk content downstream).
	multiFilePatch := "*** Begin Patch\n*** Add File: first.go\n+package main\n*** Update File: second.go\n@@\n-old\n+new\n*** End Patch\n"
	multiFileInput, err := json.Marshal(map[string]string{"command": multiFilePatch})
	if err != nil {
		t.Fatalf("marshal multiFileInput: %v", err)
	}
	if got := CodexTools.FilePath(multiFileInput); got != "first.go" {
		t.Errorf("codex multi-file FilePath should be the first section, got %q", got)
	}
	// Delete File: grammar confirmed against openai/codex's own parser
	// (codex-rs/apply-patch/src/parser.rs: "delete_hunk: \"*** Delete File: \"
	// filename LF" — no body follows). Not yet seen in a live payload, but the
	// marker text/position is from the real grammar, not a guess. codexDiff
	// should report no diff (nothing to scan), while FilePath still resolves.
	deleteInput, err := json.Marshal(map[string]string{"command": "*** Begin Patch\n*** Delete File: old.go\n*** End Patch\n"})
	if err != nil {
		t.Fatalf("marshal deleteInput: %v", err)
	}
	if got := CodexTools.FilePath(deleteInput); got != "old.go" {
		t.Errorf("codex Delete File FilePath = %q, want %q", got, "old.go")
	}
	if d := CodexTools.Changes("apply_patch", deleteInput); d != nil {
		t.Errorf("codex Delete File should yield no diff, got %+v", d)
	}
	// Move to (rename): grammar confirmed against the same source —
	// "update_hunk: ... change_move? change?", "change_move: \"*** Move to: \"
	// filename LF" — appears right after "*** Update File:", before any "@@"
	// hunks. It must be consumed as a marker, not leaked into the hunk body
	// (which would corrupt codexHunkDiffs's line-prefix parsing).
	movePatch := "*** Begin Patch\n*** Update File: old.go\n*** Move to: new.go\n@@\n-old line\n+new line\n*** End Patch\n"
	moveInput, err := json.Marshal(map[string]string{"command": movePatch})
	if err != nil {
		t.Fatalf("marshal moveInput: %v", err)
	}
	if got := CodexTools.FilePath(moveInput); got != "old.go" {
		t.Errorf("codex Move to FilePath should be the Update File source path, got %q", got)
	}
	if d := CodexTools.Changes("apply_patch", moveInput); len(d) != 1 || d[0].Before != "old line\n" || d[0].After != "new line\n" {
		t.Errorf("codex Move to hunk should parse cleanly (Move to line consumed, not leaked into body): %+v", d)
	}
	// End of File: grammar confirmed against the same source — "change:
	// (change_context | change_line)+ eof_line?", "eof_line: \"*** End of
	// File\" LF" — trails a change block's lines. Must be consumed as a
	// marker, not leaked into the hunk body as a bogus context line.
	eofPatch := "*** Begin Patch\n*** Update File: file.txt\n@@\n+quux\n*** End of File\n*** End Patch\n"
	eofInput, err := json.Marshal(map[string]string{"command": eofPatch})
	if err != nil {
		t.Fatalf("marshal eofInput: %v", err)
	}
	if d := CodexTools.Changes("apply_patch", eofInput); len(d) != 1 || d[0].Before != "" || d[0].After != "quux\n" {
		t.Errorf("codex End of File marker should be consumed, not leaked into hunk body: %+v", d)
	}
	// cursor: Write -> content, file_path key
	if got := CursorTools.FilePath(json.RawMessage(`{"file_path":"/r/Demo.java"}`)); got != "/r/Demo.java" {
		t.Errorf("cursor file path: %q", got)
	}
	if d := CursorTools.Changes("Write", json.RawMessage(`{"file_path":"/r/Demo.java","content":"class X{}"}`)); d[0].Before != "" || d[0].After != "class X{}" {
		t.Errorf("cursor write diff (content): %+v", d)
	}
	if d := CursorTools.Changes("Write", json.RawMessage(`{"path":"/r/Demo.java","contents":"class Y{}"}`)); d[0].Before != "" || d[0].After != "class Y{}" {
		t.Errorf("cursor write diff (contents): %+v", d)
	}
	if d := CursorTools.Changes("StrReplace", json.RawMessage(`{"path":"/r/Demo.java","old_string":"a","new_string":"b"}`)); d[0].Before != "a" || d[0].After != "b" {
		t.Errorf("cursor strreplace diff: %+v", d)
	}
}

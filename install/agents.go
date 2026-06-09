package install

import "path/filepath"

// InstallClaude writes Claude Code hook configuration to
// ~/.claude/settings.json. The schema is the documented Claude Code form
// (each event is a list of hook groups; an optional matcher scopes a group
// to a subset of tool names). The pre-file-write hook is matcher-scoped to
// Write|Edit|MultiEdit so it only fires on file-modifying tools.
//
// Routes wired:
//   - claude-stop                — agent finished
//   - claude-pre-tool-use        — before any tool call
//   - claude-pre-file-write      — before a write/edit/multiedit tool call (matcher-scoped)
//   - claude-user-prompt-submit  — before the user prompt is sent to the model
func InstallClaude(home string, cmdFor CmdForFunc) error {
	return PatchJSON(filepath.Join(home, ".claude", "settings.json"), func(m map[string]any) {
		hooks := EnsureMap(m, "hooks")
		hooks["Stop"] = []any{ClaudeHook(cmdFor("claude-stop"))}
		hooks["PreToolUse"] = []any{
			ClaudeHook(cmdFor("claude-pre-tool-use")),
			ClaudeHookWithMatcher(cmdFor("claude-pre-file-write"), "Write|Edit|MultiEdit"),
		}
		hooks["PostToolUse"] = []any{}
		hooks["UserPromptSubmit"] = []any{ClaudeHook(cmdFor("claude-user-prompt-submit"))}
	})
}

// InstallCursor writes Cursor hook configuration to ~/.cursor/hooks.json
// using the documented Cursor v1 schema (top-level "version" + nested "hooks"
// map; each event maps to an array of hook entries).
//
// Routes wired:
//   - cursor-stop                  — agent finished
//   - cursor-before-shell          — before a shell command runs
//   - cursor-before-mcp            — before an MCP tool runs
//   - cursor-before-file-read      — before a file read
//   - cursor-after-file-edit       — after a file edit
//   - cursor-before-submit-prompt  — before the user prompt is submitted
func InstallCursor(home string, cmdFor CmdForFunc) error {
	return PatchJSON(filepath.Join(home, ".cursor", "hooks.json"), func(m map[string]any) {
		m["version"] = 1
		hooks := EnsureMap(m, "hooks")
		hooks["beforeSubmitPrompt"] = []any{CursorHook(cmdFor("cursor-before-submit-prompt"))}
		hooks["beforeShellExecution"] = []any{CursorHook(cmdFor("cursor-before-shell"))}
		hooks["beforeMCPExecution"] = []any{CursorHook(cmdFor("cursor-before-mcp"))}
		hooks["beforeReadFile"] = []any{CursorHook(cmdFor("cursor-before-file-read"))}
		hooks["afterFileEdit"] = []any{CursorHook(cmdFor("cursor-after-file-edit"))}
		hooks["stop"] = []any{CursorHook(cmdFor("cursor-stop"))}
	})
}

// InstallWindsurf writes Windsurf Cascade hook configuration to
// ~/.codeium/windsurf/hooks.json.
//
// Routes wired:
//   - windsurf-pre-run-command       — before a shell command runs
//   - windsurf-pre-mcp-tool-use      — before an MCP tool runs
//   - windsurf-pre-user-prompt       — before the user prompt is sent
//   - windsurf-pre-write-code        — before a code write
//   - windsurf-post-cascade-response — after the agent finishes responding
func InstallWindsurf(home string, cmdFor CmdForFunc) error {
	return PatchJSON(filepath.Join(home, ".codeium", "windsurf", "hooks.json"), func(m map[string]any) {
		hooks := EnsureMap(m, "hooks")
		hooks["pre_run_command"] = []any{WindsurfHook(cmdFor("windsurf-pre-run-command"))}
		hooks["pre_mcp_tool_use"] = []any{WindsurfHook(cmdFor("windsurf-pre-mcp-tool-use"))}
		hooks["pre_user_prompt"] = []any{WindsurfHook(cmdFor("windsurf-pre-user-prompt"))}
		hooks["pre_write_code"] = []any{WindsurfHook(cmdFor("windsurf-pre-write-code"))}
		hooks["post_write_code"] = []any{}
		hooks["post_cascade_response"] = []any{WindsurfHook(cmdFor("windsurf-post-cascade-response"))}
	})
}

// InstallDroid writes Factory Droid hook configuration to
// ~/.factory/settings.json. Droid uses the same schema as Claude Code.
//
// Routes wired:
//   - droid-stop                — agent finished
//   - droid-pre-tool-use        — before any tool call
//   - droid-pre-file-write      — before a write/edit/multiedit tool call (matcher-scoped)
//   - droid-user-prompt-submit  — before the user prompt is sent
func InstallDroid(home string, cmdFor CmdForFunc) error {
	return PatchJSON(filepath.Join(home, ".factory", "settings.json"), func(m map[string]any) {
		hooks := EnsureMap(m, "hooks")
		hooks["Stop"] = []any{ClaudeHook(cmdFor("droid-stop"))}
		hooks["PreToolUse"] = []any{
			ClaudeHook(cmdFor("droid-pre-tool-use")),
			ClaudeHookWithMatcher(cmdFor("droid-pre-file-write"), "Write|Edit|MultiEdit"),
		}
		hooks["PostToolUse"] = []any{}
		hooks["UserPromptSubmit"] = []any{ClaudeHook(cmdFor("droid-user-prompt-submit"))}
	})
}

// InstallGemini writes Gemini CLI hook configuration to
// ~/.gemini/settings.json.
//
// Routes wired:
//   - gemini-before-agent      — agent starting
//   - gemini-before-tool       — before any tool call
//   - gemini-before-file-tool  — before a file-write tool call
//   - gemini-after-agent       — agent finished
func InstallGemini(home string, cmdFor CmdForFunc) error {
	return PatchJSON(filepath.Join(home, ".gemini", "settings.json"), func(m map[string]any) {
		hooks := EnsureMap(m, "hooks")
		hooks["BeforeAgent"] = []any{GeminiHook(cmdFor("gemini-before-agent"))}
		hooks["BeforeTool"] = []any{
			GeminiHook(cmdFor("gemini-before-tool")),
			GeminiHook(cmdFor("gemini-before-file-tool")),
		}
		hooks["AfterTool"] = []any{}
		hooks["AfterAgent"] = []any{GeminiHook(cmdFor("gemini-after-agent"))}
	})
}

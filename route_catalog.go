package agenthooks

// HookStyle describes how a platform's settings file encodes a hook entry.
type HookStyle string

const (
	// StyleClaudeNested writes {"hooks": {EventKey: [{"type":"command","command":...}]}}.
	// Used by Claude Code and Factory Droid.
	StyleClaudeNested HookStyle = "claude-nested"
	// StyleFlatCommand writes {EventKey: {"command": ...}} at the top level.
	// Used by Cursor and Windsurf Cascade.
	StyleFlatCommand HookStyle = "flat-command"
	// StyleGeminiNested writes {"hooks": {EventKey: [{"matcher":"","hooks":[{"type":"command","command":...}]}]}}.
	// Used by Gemini CLI.
	StyleGeminiNested HookStyle = "gemini-nested"
)

// CatalogEntry maps one unified hook route to the settings-file slot a platform
// expects it in. It is the single source of truth shared by route registration
// (the WhenAgentIdle / BeforeToolCall / ... functions) and `agenthooks install`.
type CatalogEntry struct {
	Agent       AgentID   // which platform this route belongs to
	Route       string    // the route name passed as the hook binary's first argument
	SettingsRel string    // settings-file path relative to the user's home directory (slash-separated)
	EventKey    string    // key under which the hook is registered in the settings file
	Style       HookStyle // how the entry is encoded in that file
}

// Catalog lists every unified route and where each platform expects it installed.
//
// Copilot is intentionally absent: VS Code Copilot hooks are registered per-project
// in .github/hooks/*.json (not a home-directory settings file), so `agenthooks install`
// does not write them automatically — see the README for manual Copilot setup.
var Catalog = []CatalogEntry{
	// Claude Code → ~/.claude/settings.json
	{AgentClaude, "claude-stop", ".claude/settings.json", "Stop", StyleClaudeNested},
	{AgentClaude, "claude-pre-tool-use", ".claude/settings.json", "PreToolUse", StyleClaudeNested},
	{AgentClaude, "claude-after-file-write", ".claude/settings.json", "PostToolUse", StyleClaudeNested},
	{AgentClaude, "claude-user-prompt-submit", ".claude/settings.json", "UserPromptSubmit", StyleClaudeNested},
	{AgentClaude, "claude-subagent-stop", ".claude/settings.json", "SubagentStop", StyleClaudeNested},
	{AgentClaude, "claude-post-tool-use-failure", ".claude/settings.json", "PostToolUseFailure", StyleClaudeNested},

	// Factory Droid → ~/.factory/settings.json
	{AgentDroid, "droid-stop", ".factory/settings.json", "Stop", StyleClaudeNested},
	{AgentDroid, "droid-pre-tool-use", ".factory/settings.json", "PreToolUse", StyleClaudeNested},
	{AgentDroid, "droid-after-file-write", ".factory/settings.json", "PostToolUse", StyleClaudeNested},
	{AgentDroid, "droid-user-prompt-submit", ".factory/settings.json", "UserPromptSubmit", StyleClaudeNested},
	{AgentDroid, "droid-subagent-stop", ".factory/settings.json", "SubagentStop", StyleClaudeNested},

	// Cursor → ~/.cursor/hooks.json
	{AgentCursor, "cursor-stop", ".cursor/hooks.json", "stop", StyleFlatCommand},
	{AgentCursor, "cursor-before-shell", ".cursor/hooks.json", "beforeShellExecution", StyleFlatCommand},
	{AgentCursor, "cursor-before-mcp", ".cursor/hooks.json", "beforeMCPExecution", StyleFlatCommand},
	{AgentCursor, "cursor-after-file-edit", ".cursor/hooks.json", "afterFileEdit", StyleFlatCommand},
	{AgentCursor, "cursor-before-submit-prompt", ".cursor/hooks.json", "beforeSubmitPrompt", StyleFlatCommand},
	{AgentCursor, "cursor-before-read-file", ".cursor/hooks.json", "beforeReadFile", StyleFlatCommand},
	{AgentCursor, "cursor-subagent-stop", ".cursor/hooks.json", "subagentStop", StyleFlatCommand},
	{AgentCursor, "cursor-post-tool-use-failure", ".cursor/hooks.json", "postToolUseFailure", StyleFlatCommand},

	// Windsurf Cascade → ~/.codeium/windsurf/hooks.json
	{AgentWindsurf, "windsurf-pre-run-command", ".codeium/windsurf/hooks.json", "pre_run_command", StyleFlatCommand},
	{AgentWindsurf, "windsurf-pre-mcp-tool-use", ".codeium/windsurf/hooks.json", "pre_mcp_tool_use", StyleFlatCommand},
	{AgentWindsurf, "windsurf-pre-user-prompt", ".codeium/windsurf/hooks.json", "pre_user_prompt", StyleFlatCommand},
	{AgentWindsurf, "windsurf-pre-read-code", ".codeium/windsurf/hooks.json", "pre_read_code", StyleFlatCommand},
	{AgentWindsurf, "windsurf-post-write-code", ".codeium/windsurf/hooks.json", "post_write_code", StyleFlatCommand},
	{AgentWindsurf, "windsurf-post-cascade-response", ".codeium/windsurf/hooks.json", "post_cascade_response", StyleFlatCommand},

	// Gemini CLI → ~/.gemini/settings.json
	{AgentGemini, "gemini-before-tool", ".gemini/settings.json", "BeforeTool", StyleGeminiNested},
	{AgentGemini, "gemini-after-agent", ".gemini/settings.json", "AfterAgent", StyleGeminiNested},
	{AgentGemini, "gemini-before-agent", ".gemini/settings.json", "BeforeAgent", StyleGeminiNested},
	{AgentGemini, "gemini-after-file-tool", ".gemini/settings.json", "AfterTool", StyleGeminiNested},
}

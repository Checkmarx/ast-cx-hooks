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
	// StyleCopilotCLINested writes {"version":1,"hooks":{EventKey:[{"type":"command","command":...}]}}.
	// Used by GitHub Copilot CLI (a dedicated hooks file under ~/.copilot/hooks/).
	StyleCopilotCLINested HookStyle = "copilot-cli-nested"
	// StyleCodexNested writes {"hooks":{EventKey:[{"matcher":"","hooks":[{"type":"command","command":...}]}]}}.
	// Used by OpenAI Codex CLI (~/.codex/hooks.json). Byte-identical to
	// StyleGeminiNested today, but kept as a distinct constant — the same
	// precedent as StyleCopilotCLINested being kept distinct from
	// StyleClaudeNested — so Codex's encoding can diverge later (e.g. optional
	// timeout/async fields) without touching Gemini's code path.
	StyleCodexNested HookStyle = "codex-nested"
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
// The VS Code Copilot extension is intentionally absent: its hooks are registered
// per-project in .github/hooks/*.json (not a home-directory settings file), so
// `agenthooks install` does not write them automatically — see the README for
// manual setup. The GitHub Copilot CLI (a distinct product) IS installable via its
// user-level ~/.copilot/hooks/ directory and is listed below.
var Catalog = []CatalogEntry{
	// Claude Code → ~/.claude/settings.json
	{AgentClaude, "claude-stop", ".claude/settings.json", "Stop", StyleClaudeNested},
	{AgentClaude, "claude-pre-tool-use", ".claude/settings.json", "PreToolUse", StyleClaudeNested},
	{AgentClaude, "claude-pre-file-write", ".claude/settings.json", "PreToolUse", StyleClaudeNested},
	{AgentClaude, "claude-after-file-write", ".claude/settings.json", "PostToolUse", StyleClaudeNested},
	{AgentClaude, "claude-user-prompt-submit", ".claude/settings.json", "UserPromptSubmit", StyleClaudeNested},
	{AgentClaude, "claude-subagent-stop", ".claude/settings.json", "SubagentStop", StyleClaudeNested},
	{AgentClaude, "claude-post-tool-use-failure", ".claude/settings.json", "PostToolUseFailure", StyleClaudeNested},

	// Factory Droid → ~/.factory/settings.json
	{AgentDroid, "droid-stop", ".factory/settings.json", "Stop", StyleClaudeNested},
	{AgentDroid, "droid-pre-tool-use", ".factory/settings.json", "PreToolUse", StyleClaudeNested},
	{AgentDroid, "droid-pre-file-write", ".factory/settings.json", "PreToolUse", StyleClaudeNested},
	{AgentDroid, "droid-after-file-write", ".factory/settings.json", "PostToolUse", StyleClaudeNested},
	{AgentDroid, "droid-user-prompt-submit", ".factory/settings.json", "UserPromptSubmit", StyleClaudeNested},
	{AgentDroid, "droid-subagent-stop", ".factory/settings.json", "SubagentStop", StyleClaudeNested},

	// Cursor → ~/.cursor/hooks.json
	{AgentCursor, "cursor-stop", ".cursor/hooks.json", "stop", StyleFlatCommand},
	{AgentCursor, "cursor-before-shell", ".cursor/hooks.json", "beforeShellExecution", StyleFlatCommand},
	{AgentCursor, "cursor-before-mcp", ".cursor/hooks.json", "beforeMCPExecution", StyleFlatCommand},
	{AgentCursor, "cursor-before-file-write", ".cursor/hooks.json", "preToolUse", StyleFlatCommand},
	{AgentCursor, "cursor-after-file-edit", ".cursor/hooks.json", "postToolUse", StyleFlatCommand},
	{AgentCursor, "cursor-before-submit-prompt", ".cursor/hooks.json", "beforeSubmitPrompt", StyleFlatCommand},
	{AgentCursor, "cursor-before-file-read", ".cursor/hooks.json", "beforeReadFile", StyleFlatCommand},
	{AgentCursor, "cursor-before-read-file", ".cursor/hooks.json", "beforeReadFile", StyleFlatCommand},
	{AgentCursor, "cursor-subagent-stop", ".cursor/hooks.json", "subagentStop", StyleFlatCommand},
	{AgentCursor, "cursor-post-tool-use-failure", ".cursor/hooks.json", "postToolUseFailure", StyleFlatCommand},

	// Windsurf Cascade → ~/.codeium/windsurf/hooks.json
	{AgentWindsurf, "windsurf-pre-run-command", ".codeium/windsurf/hooks.json", "pre_run_command", StyleFlatCommand},
	{AgentWindsurf, "windsurf-pre-mcp-tool-use", ".codeium/windsurf/hooks.json", "pre_mcp_tool_use", StyleFlatCommand},
	{AgentWindsurf, "windsurf-pre-user-prompt", ".codeium/windsurf/hooks.json", "pre_user_prompt", StyleFlatCommand},
	{AgentWindsurf, "windsurf-pre-read-code", ".codeium/windsurf/hooks.json", "pre_read_code", StyleFlatCommand},
	{AgentWindsurf, "windsurf-pre-write-code", ".codeium/windsurf/hooks.json", "pre_write_code", StyleFlatCommand},
	{AgentWindsurf, "windsurf-post-write-code", ".codeium/windsurf/hooks.json", "post_write_code", StyleFlatCommand},
	{AgentWindsurf, "windsurf-post-cascade-response", ".codeium/windsurf/hooks.json", "post_cascade_response", StyleFlatCommand},

	// Gemini CLI → ~/.gemini/settings.json
	{AgentGemini, "gemini-before-tool", ".gemini/settings.json", "BeforeTool", StyleGeminiNested},
	{AgentGemini, "gemini-before-file-tool", ".gemini/settings.json", "BeforeTool", StyleGeminiNested},
	{AgentGemini, "gemini-after-agent", ".gemini/settings.json", "AfterAgent", StyleGeminiNested},
	{AgentGemini, "gemini-before-agent", ".gemini/settings.json", "BeforeAgent", StyleGeminiNested},
	{AgentGemini, "gemini-after-file-tool", ".gemini/settings.json", "AfterTool", StyleGeminiNested},

	// GitHub Copilot CLI → ~/.copilot/hooks/agenthooks.json (PascalCase event keys
	// select the VS Code-compatible snake_case payload format).
	{AgentCopilotCLI, "copilot-cli-stop", ".copilot/hooks/agenthooks.json", "Stop", StyleCopilotCLINested},
	{AgentCopilotCLI, "copilot-cli-pre-tool-use", ".copilot/hooks/agenthooks.json", "PreToolUse", StyleCopilotCLINested},
	{AgentCopilotCLI, "copilot-cli-pre-file-write", ".copilot/hooks/agenthooks.json", "PreToolUse", StyleCopilotCLINested},
	{AgentCopilotCLI, "copilot-cli-after-file-write", ".copilot/hooks/agenthooks.json", "PostToolUse", StyleCopilotCLINested},
	{AgentCopilotCLI, "copilot-cli-post-tool-use-failure", ".copilot/hooks/agenthooks.json", "PostToolUseFailure", StyleCopilotCLINested},
	{AgentCopilotCLI, "copilot-cli-subagent-stop", ".copilot/hooks/agenthooks.json", "SubagentStop", StyleCopilotCLINested},
	{AgentCopilotCLI, "copilot-cli-user-prompt-submit", ".copilot/hooks/agenthooks.json", "UserPromptSubmit", StyleCopilotCLINested},

	// OpenAI Codex CLI → ~/.codex/hooks.json
	{AgentCodex, "codex-stop", ".codex/hooks.json", "Stop", StyleCodexNested},
	{AgentCodex, "codex-pre-tool-use", ".codex/hooks.json", "PreToolUse", StyleCodexNested},
	{AgentCodex, "codex-pre-file-write", ".codex/hooks.json", "PreToolUse", StyleCodexNested},
	{AgentCodex, "codex-after-file-write", ".codex/hooks.json", "PostToolUse", StyleCodexNested},
	{AgentCodex, "codex-user-prompt-submit", ".codex/hooks.json", "UserPromptSubmit", StyleCodexNested},
	{AgentCodex, "codex-subagent-stop", ".codex/hooks.json", "SubagentStop", StyleCodexNested},
}

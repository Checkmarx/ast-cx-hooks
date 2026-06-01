// Package copilot provides types and response helpers for GitHub Copilot
// agent hooks in Visual Studio Code.
//
// VS Code Copilot agent hooks are configured via JSON files at:
//   - Workspace scope: .github/hooks/*.json (also reads .claude/settings.json)
//   - User scope:      ~/.copilot/hooks    (also reads ~/.claude/settings.json)
//   - Per-agent:       hooks field in .agent.md frontmatter
//
// Each hook process receives a JSON payload on stdin and writes a JSON
// decision to stdout. The wire protocol is closely modeled on Claude Code's
// hooks; the main difference is that Copilot uses camelCase keys for
// sessionId and hookEventName.
//
// Supported events: SessionStart, UserPromptSubmit, PreToolUse, PostToolUse,
// PreCompact, SubagentStart, SubagentStop, Stop.
//
// See https://code.visualstudio.com/docs/copilot/customization/hooks for the
// official reference. This package targets the Preview release; field shapes
// may shift before GA.
package copilot

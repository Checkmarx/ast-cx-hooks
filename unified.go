package agenthooks

import (
	"github.com/CheckmarxDev/ast-cx-hooks/claude"
	"github.com/CheckmarxDev/ast-cx-hooks/copilot"
	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
	"github.com/CheckmarxDev/ast-cx-hooks/droid"
	"github.com/CheckmarxDev/ast-cx-hooks/gemini"
	"github.com/CheckmarxDev/ast-cx-hooks/windsurf"
)

// This file is the unified-handler registry. Each function registers one event
// category across every platform that supports it; the per-platform translation
// between wire types and the unified hookcore vocabulary lives in each platform
// package's adapters.go (so the logic sits next to that platform's types).

// WhenAgentIdle registers a unified handler for "agent finished responding" events:
//   - Claude Code     → "claude-stop"
//   - Cursor          → "cursor-stop"
//   - Windsurf        → "windsurf-post-cascade-response" (fire-and-forget; Interrupt logged, ignored)
//   - Factory Droid   → "droid-stop"
//   - Gemini CLI      → "gemini-after-agent"
//   - VS Code Copilot → "copilot-stop"
func WhenAgentIdle(fn AgentIdleFunc) {
	registerAdapters(fn,
		claude.IdleAdapter, cursor.IdleAdapter, windsurf.IdleAdapter,
		droid.IdleAdapter, gemini.IdleAdapter, copilot.IdleAdapter,
	)
}

// BeforeToolCall registers a unified handler for pre-execution events:
//   - Claude Code     → "claude-pre-tool-use"
//   - Cursor          → "cursor-before-shell", "cursor-before-mcp"
//   - Windsurf        → "windsurf-pre-run-command", "windsurf-pre-mcp-tool-use" (blocking via exit 2)
//   - Factory Droid   → "droid-pre-tool-use" (blocking via exit 2)
//   - Gemini CLI      → "gemini-before-tool"
//   - VS Code Copilot → "copilot-pre-tool-use"
func BeforeToolCall(fn ToolCallFunc) {
	registerAdapters(fn,
		claude.ToolAdapter, cursor.ShellToolAdapter, cursor.MCPToolAdapter,
		windsurf.RunCommandAdapter, windsurf.MCPToolAdapter,
		droid.ToolAdapter, gemini.ToolAdapter, copilot.ToolAdapter,
	)
}

// AfterFileWrite registers a unified handler for post-file-edit events:
//   - Claude Code     → "claude-after-file-write"
//   - Cursor          → "cursor-after-file-edit"   (fire-and-forget)
//   - Windsurf        → "windsurf-post-write-code"  (fire-and-forget)
//   - Factory Droid   → "droid-after-file-write"
//   - Gemini CLI      → "gemini-after-file-tool"
//   - VS Code Copilot → "copilot-after-file-write"
func AfterFileWrite(fn FileWriteFunc) {
	registerAdapters(fn,
		claude.FileWriteAdapter, cursor.FileWriteAdapter, windsurf.FileWriteAdapter,
		droid.FileWriteAdapter, gemini.FileWriteAdapter, copilot.FileWriteAdapter,
	)
}

// BeforePrompt registers a unified handler for prompt-submission events:
//   - Claude Code     → "claude-user-prompt-submit"
//   - Cursor          → "cursor-before-submit-prompt"
//   - Windsurf        → "windsurf-pre-user-prompt"   (blocking via exit 2)
//   - Factory Droid   → "droid-user-prompt-submit"
//   - Gemini CLI      → "gemini-before-agent"
//   - VS Code Copilot → "copilot-user-prompt-submit"
func BeforePrompt(fn PromptFunc) {
	registerAdapters(fn,
		claude.PromptAdapter, cursor.PromptAdapter, windsurf.PromptAdapter,
		droid.PromptAdapter, gemini.PromptAdapter, copilot.PromptAdapter,
	)
}

// WhenSubagentIdle registers a unified handler for "subagent finished" events.
// It reuses AgentIdleEvent/IdleVerdict; Interrupt blocks the subagent from stopping
// (on Cursor it auto-submits a follow-up):
//   - Claude Code     → "claude-subagent-stop"
//   - Factory Droid   → "droid-subagent-stop"
//   - VS Code Copilot → "copilot-subagent-stop"
//   - Cursor          → "cursor-subagent-stop"
func WhenSubagentIdle(fn AgentIdleFunc) {
	registerAdapters(fn,
		claude.SubagentIdleAdapter, droid.SubagentIdleAdapter,
		copilot.SubagentIdleAdapter, cursor.SubagentIdleAdapter,
	)
}

// AfterToolFailure registers a unified handler for failed tool calls:
//   - Claude Code → "claude-post-tool-use-failure" (can block / annotate)
//   - Cursor      → "cursor-post-tool-use-failure" (observational; verdict ignored)
func AfterToolFailure(fn ToolFailureFunc) {
	registerAdapters(fn, claude.ToolFailureAdapter, cursor.ToolFailureAdapter)
}

// BeforeFileRead registers a unified handler for "agent about to read a file" events:
//   - Cursor   → "cursor-before-read-file" (allow/deny)
//   - Windsurf → "windsurf-pre-read-code"  (blocking via exit 2)
func BeforeFileRead(fn FileReadFunc) {
	registerAdapters(fn, cursor.FileReadAdapter, windsurf.FileReadAdapter)
}

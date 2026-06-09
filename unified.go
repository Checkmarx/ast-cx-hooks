package agenthooks

// This file declares the 8 unified hooks. Each registers a single DEFAULT handler
// across every platform that supports it; to run several independent
// implementations behind one hook (e.g. per team), register named scenarios with
// the matching <Hook>Scenario function (see scenarios.go). The per-platform
// translation between wire types and the unified vocabulary lives in each platform
// package's adapters.go.

// WhenAgentIdle registers the default handler for "agent finished responding" events:
//   - Claude Code     → "claude-stop"
//   - Cursor          → "cursor-stop"
//   - Windsurf        → "windsurf-post-cascade-response" (fire-and-forget; Interrupt logged, ignored)
//   - Factory Droid   → "droid-stop"
//   - Gemini CLI      → "gemini-after-agent"
//   - VS Code Copilot → "copilot-stop"
func WhenAgentIdle(fn AgentIdleFunc) { idleReg.setDefault(fn) }

// WhenSubagentIdle registers the default handler for "subagent finished" events.
// It reuses AgentIdleEvent/IdleVerdict; Interrupt blocks the subagent from stopping
// (on Cursor it auto-submits a follow-up):
//   - Claude Code     → "claude-subagent-stop"
//   - Factory Droid   → "droid-subagent-stop"
//   - VS Code Copilot → "copilot-subagent-stop"
//   - Cursor          → "cursor-subagent-stop"
func WhenSubagentIdle(fn AgentIdleFunc) { subagentIdleReg.setDefault(fn) }

// BeforeToolCall registers the default handler for pre-execution events:
//   - Claude Code     → "claude-pre-tool-use"
//   - Cursor          → "cursor-before-shell", "cursor-before-mcp"
//   - Windsurf        → "windsurf-pre-run-command", "windsurf-pre-mcp-tool-use" (blocking via exit 2)
//   - Factory Droid   → "droid-pre-tool-use" (blocking via exit 2)
//   - Gemini CLI      → "gemini-before-tool"
//   - VS Code Copilot → "copilot-pre-tool-use"
func BeforeToolCall(fn ToolCallFunc) { toolCallReg.setDefault(fn) }

// AfterToolFailure registers the default handler for failed tool calls:
//   - Claude Code → "claude-post-tool-use-failure" (can block / annotate)
//   - Cursor      → "cursor-post-tool-use-failure" (observational; verdict ignored)
func AfterToolFailure(fn ToolFailureFunc) { toolFailureReg.setDefault(fn) }

// AfterFileWrite registers the default handler for post-file-edit events:
//   - Claude Code     → "claude-after-file-write"
//   - Cursor          → "cursor-after-file-write" (postToolUse; delivers additional_context, cannot block)
//   - Windsurf        → "windsurf-post-write-code"  (fire-and-forget)
//   - Factory Droid   → "droid-after-file-write"
//   - Gemini CLI      → "gemini-after-file-tool"
//   - VS Code Copilot → "copilot-after-file-write"
func AfterFileWrite(fn FileWriteFunc) { fileWriteReg.setDefault(fn) }

// BeforeFileEdit registers the default handler for "agent about to write/edit a
// file" events — a pre-write GATE that can BLOCK the change before it lands (unlike
// AfterFileWrite, which fires post-write and can only annotate). The handler sees
// the proposed FilePath/Changes and returns a PreToolUse-style verdict
// (AcceptEdit / RejectEdit / RejectEditWithContext / AskBeforeEdit):
//   - Claude Code   → "claude-pre-file-write" (PreToolUse, Write/Edit/MultiEdit; deny + additionalContext)
//   - Factory Droid → "droid-pre-file-write"  (PreToolUse, Write/Edit; deny only — no context channel)
func BeforeFileEdit(fn FileEditFunc) { fileEditReg.setDefault(fn) }

// BeforeFileRead registers the default handler for "agent about to read a file" events:
//   - Cursor   → "cursor-before-read-file" (allow/deny)
//   - Windsurf → "windsurf-pre-read-code"  (blocking via exit 2)
func BeforeFileRead(fn FileReadFunc) { fileReadReg.setDefault(fn) }

// BeforePrompt registers the default handler for prompt-submission events:
//   - Claude Code     → "claude-user-prompt-submit"
//   - Cursor          → "cursor-before-submit-prompt"
//   - Windsurf        → "windsurf-pre-user-prompt"   (blocking via exit 2)
//   - Factory Droid   → "droid-user-prompt-submit"
//   - Gemini CLI      → "gemini-before-agent"
//   - VS Code Copilot → "copilot-user-prompt-submit"
func BeforePrompt(fn PromptFunc) { promptReg.setDefault(fn) }

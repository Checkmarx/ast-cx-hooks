package claude

import "github.com/Checkmarx/ast-cx-hooks/internal/hookcore"

// This file owns the translation between Claude's wire types and the unified
// hookcore vocabulary. The root agenthooks package wires these adapters into a
// registry; the per-platform logic lives here for locality.

// IdleAdapter handles the unified "agent finished" hook for Claude (Stop).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "claude-stop", func() {
		hookcore.Run(func(ev StopEvent) StopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentClaude, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if v.Proceed {
				return LetStop()
			}
			return HaltAndContinue(v.Feedback)
		})
	}
}

// SubagentIdleAdapter handles the unified "subagent finished" hook for Claude.
func SubagentIdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "claude-subagent-stop", func() {
		hookcore.Run(func(ev SubagentStopEvent) SubagentStopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentClaude, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if v.Proceed {
				return LetSubagentStop()
			}
			return HaltSubagent(v.Feedback)
		})
	}
}

// preToolDecision maps a unified PreToolUse-style verdict to Claude's
// PreToolUseResult, honoring the full surface (allow/deny/ask + additionalContext
// + updatedInput) via the shared Path() classifier. It is the single seam both
// PreToolUse gates use: the tool-call gate (ToolAdapter) and the pre-file-write
// gate (FileEditAdapter).
func preToolDecision(v hookcore.ToolVerdict) PreToolUseResult {
	switch v.Path() {
	case hookcore.PathAllowWithInput:
		return ApproveToolUseWithInput(v.RewrittenInput)
	case hookcore.PathAllowWithContext:
		return ApproveToolUseWithContext(v.Context)
	case hookcore.PathAllowWithNote:
		return ApproveToolUseWithNote(v.Message)
	case hookcore.PathAsk:
		return AskUserAboutTool(v.Message)
	case hookcore.PathDenyWithContext:
		return DenyToolUseWithContext(v.Message, v.Context)
	case hookcore.PathDeny:
		return DenyToolUse(v.Message)
	default: // PathAllow
		return ApproveToolUse()
	}
}

// ToolAdapter handles the unified pre-tool-call hook for Claude (Bash + mcp__*).
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "claude-pre-tool-use", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			kind, cmd := hookcore.ClaudeTools.Kind(ev.ToolName, ev.ToolInput)
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentClaude, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileEditAdapter handles the unified pre-file-write GATE for Claude: PreToolUse
// scoped to the file-writing tools (Write/Edit/MultiEdit). Unlike FileWriteAdapter
// (PostToolUse, after the write has landed), this fires BEFORE the write and can
// DENY it — the route a security scanner uses to block a vulnerable change before
// it reaches disk, optionally injecting remediation context on the deny. Non-write
// tools are approved untouched so the generic claude-pre-tool-use gate owns them.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "claude-pre-file-write", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			if !hookcore.ClaudeTools.IsWrite(ev.ToolName) {
				return ApproveToolUse()
			}
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentClaude, SessionID: ev.SessionID,
				FilePath: hookcore.ClaudeTools.FilePath(ev.ToolInput),
				Changes:  hookcore.ClaudeTools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Claude (PostToolUse Write/Edit).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "claude-after-file-write", func() {
		hookcore.Run(func(ev PostToolUseEvent) PostToolUseResult {
			if !hookcore.ClaudeTools.IsWrite(ev.ToolName) {
				return AcknowledgeToolUse()
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentClaude, SessionID: ev.SessionID,
				FilePath: hookcore.ClaudeTools.FilePath(ev.ToolInput),
				Changes:  hookcore.ClaudeTools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			if v.Reject {
				if v.Context != "" {
					return RejectToolResultWithContext(v.Feedback, v.Context)
				}
				return RejectToolResult(v.Feedback)
			}
			if v.Footnote != "" {
				return AddToolContext(v.Footnote)
			}
			return AcknowledgeToolUse()
		})
	}
}

// PromptAdapter handles the unified prompt-submit hook for Claude.
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "claude-user-prompt-submit", func() {
		hookcore.Run(func(ev UserPromptSubmitEvent) UserPromptSubmitResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentClaude, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
			})
			if !v.Accept {
				return RejectPrompt(v.Message)
			}
			if v.Message != "" {
				return AppendToPrompt(v.Message)
			}
			return ApprovePrompt()
		})
	}
}

// ToolFailureAdapter handles the unified post-tool-failure hook for Claude.
func ToolFailureAdapter(fn hookcore.ToolFailureFunc) (string, func()) {
	return "claude-post-tool-use-failure", func() {
		hookcore.Run(func(ev PostToolUseFailureEvent) PostToolUseFailureResult {
			v := fn(hookcore.ToolFailureEvent{
				Agent: hookcore.AgentClaude, SessionID: ev.SessionID, ToolName: ev.ToolName,
				Error: ev.Error, WorkDir: ev.WorkDir, Raw: &ev,
			})
			if v.Reject {
				return RejectAfterFailure(v.Feedback)
			}
			if v.Footnote != "" {
				return AnnotateToolFailure(v.Footnote)
			}
			return AcknowledgeToolFailure()
		})
	}
}

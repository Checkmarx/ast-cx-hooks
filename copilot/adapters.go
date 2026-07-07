package copilot

import "github.com/Checkmarx/ast-cx-hooks/internal/hookcore"

// This file owns the translation between VS Code Copilot's wire types and the
// unified hookcore vocabulary. The root agenthooks package wires these adapters
// into a registry; the per-platform logic lives here for locality.

// IdleAdapter handles the unified "agent finished" hook for Copilot (Stop).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "copilot-stop", func() {
		hookcore.Run(func(ev StopEvent) StopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCopilot, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if v.Proceed {
				return LetStop()
			}
			return HaltAndContinue(v.Feedback)
		})
	}
}

// SubagentIdleAdapter handles the unified "subagent finished" hook for Copilot.
func SubagentIdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "copilot-subagent-stop", func() {
		hookcore.Run(func(ev SubagentStopEvent) SubagentStopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCopilot, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if v.Proceed {
				return LetSubagentStop()
			}
			return KeepSubagentRunning(v.Feedback)
		})
	}
}

// preToolDecision maps a unified PreToolUse-style verdict to Copilot's
// PreToolUseResult — the full surface (allow/deny/ask + additionalContext +
// updatedInput) via the shared Path() classifier. Used by both PreToolUse gates:
// the tool-call gate (ToolAdapter) and the pre-file-write gate (FileEditAdapter).
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

// ToolAdapter handles the unified pre-tool-call hook for Copilot (PreToolUse).
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "copilot-pre-tool-use", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			kind, cmd := hookcore.CopilotTools.Kind(ev.ToolName, ev.ToolInput)
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCopilot, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileEditAdapter handles the unified pre-file-write GATE for Copilot: PreToolUse
// scoped to the file-writing tools (createFile/editFiles). Fires BEFORE the write
// and can DENY it (with additionalContext); non-write tools are approved untouched
// so the generic copilot-pre-tool-use gate owns them.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "copilot-pre-file-write", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			if !hookcore.CopilotTools.IsWrite(ev.ToolName) {
				return ApproveToolUse()
			}
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentCopilot, SessionID: ev.SessionID,
				FilePath: hookcore.CopilotTools.FilePath(ev.ToolInput),
				Changes:  hookcore.CopilotTools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Copilot (PostToolUse Write/Edit).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "copilot-after-file-write", func() {
		hookcore.Run(func(ev PostToolUseEvent) PostToolUseResult {
			if !hookcore.CopilotTools.IsWrite(ev.ToolName) {
				return AcknowledgeToolUse()
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentCopilot, SessionID: ev.SessionID,
				FilePath: hookcore.CopilotTools.FilePath(ev.ToolInput),
				Changes:  hookcore.CopilotTools.Changes(ev.ToolName, ev.ToolInput),
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

// PromptAdapter handles the unified prompt-submit hook for Copilot.
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "copilot-user-prompt-submit", func() {
		hookcore.Run(func(ev UserPromptSubmitEvent) UserPromptSubmitResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentCopilot, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
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

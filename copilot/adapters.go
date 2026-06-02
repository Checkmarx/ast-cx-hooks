package copilot

import "github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"

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

// ToolAdapter handles the unified pre-tool-call hook for Copilot (PreToolUse).
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "copilot-pre-tool-use", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			kind, cmd := hookcore.StandardToolKind(ev.ToolName, ev.ToolInput)
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCopilot, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			if v.Permit {
				if v.RewrittenInput != nil {
					return ApproveToolUseWithInput(v.RewrittenInput)
				}
				if v.Message != "" {
					return ApproveToolUseWithNote(v.Message)
				}
				return ApproveToolUse()
			}
			if v.NeedsConfirm {
				return AskUserAboutTool(v.Message)
			}
			return DenyToolUse(v.Message)
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Copilot (PostToolUse Write/Edit).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "copilot-after-file-write", func() {
		hookcore.Run(func(ev PostToolUseEvent) PostToolUseResult {
			if !hookcore.IsStandardWriteTool(ev.ToolName) {
				return AcknowledgeToolUse()
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentCopilot, SessionID: ev.SessionID,
				FilePath: hookcore.StandardFilePath(ev.ToolInput),
				Changes:  hookcore.StandardWriteChanges(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			if v.Reject {
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

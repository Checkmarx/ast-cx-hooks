package cursor

import "github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"

// This file owns the translation between Cursor's wire types and the unified
// hookcore vocabulary. The root agenthooks package wires these adapters into a
// registry; the per-platform logic lives here for locality.

// permissionResult maps a unified ToolVerdict onto Cursor's PermissionResult.
// Cursor does not support input rewriting on these gates, so RewrittenInput is
// ignored here.
func permissionResult(v hookcore.ToolVerdict) PermissionResult {
	if v.Permit {
		if v.Message != "" {
			return PermitWithNote(v.Message)
		}
		return Permit()
	}
	if v.NeedsConfirm {
		return RequestConfirmation(v.Message, v.Message)
	}
	return Forbid(v.Message, v.Message)
}

// IdleAdapter handles the unified "agent finished" hook for Cursor (stop).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "cursor-stop", func() {
		hookcore.Run(func(ev StopEvent) StopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				CompletionStatus: ev.Status, AutoRetryCount: ev.LoopCount, Raw: &ev,
			})
			if v.Proceed {
				return LetStop()
			}
			return SendFollowup(v.Feedback)
		})
	}
}

// SubagentIdleAdapter handles the unified "subagent finished" hook for Cursor.
func SubagentIdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "cursor-subagent-stop", func() {
		hookcore.Run(func(ev SubagentStopEvent) SubagentStopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				CompletionStatus: ev.Status, AutoRetryCount: ev.LoopCount, Raw: &ev,
			})
			if v.Proceed {
				return LetSubagentStop()
			}
			return SendSubagentFollowup(v.Feedback)
		})
	}
}

// ShellToolAdapter handles the unified pre-tool-call hook for Cursor shell executions.
func ShellToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "cursor-before-shell", func() {
		hookcore.Run(func(ev ShellPreEvent) ShellPreResult {
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCursor, Kind: hookcore.ToolKindShell,
				Command: ev.Command, WorkDir: ev.WorkDir, Raw: &ev,
			})
			return permissionResult(v)
		})
	}
}

// MCPToolAdapter handles the unified pre-tool-call hook for Cursor MCP executions.
func MCPToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "cursor-before-mcp", func() {
		hookcore.Run(func(ev MCPPreEvent) MCPPreResult {
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCursor, Kind: hookcore.ToolKindMCP,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput,
				ServerURL: ev.ServerURL, Command: ev.Command, Raw: &ev,
			})
			return permissionResult(v)
		})
	}
}

// FileWriteAdapter handles the unified post-file-edit hook for Cursor (afterFileEdit).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "cursor-after-file-edit", func() {
		hookcore.Run(func(ev FileEditEvent) FileEditResult {
			changes := make([]hookcore.FileDiff, len(ev.Edits))
			for i, e := range ev.Edits {
				changes[i] = hookcore.FileDiff{Before: e.OldText, After: e.NewText}
			}
			fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: ev.FilePath, Changes: changes, Raw: &ev,
			})
			return FileEditResult{}
		})
	}
}

// PromptAdapter handles the unified prompt-submit hook for Cursor.
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "cursor-before-submit-prompt", func() {
		hookcore.Run(func(ev PromptPreEvent) PromptPreResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				Text: ev.Prompt, Raw: &ev,
			})
			if !v.Accept {
				return BlockPrompt(v.Message)
			}
			return AcceptPrompt()
		})
	}
}

// ToolFailureAdapter handles the unified post-tool-failure hook for Cursor (observational).
func ToolFailureAdapter(fn hookcore.ToolFailureFunc) (string, func()) {
	return "cursor-post-tool-use-failure", func() {
		hookcore.Run(func(ev ToolFailureEvent) ToolFailureResult {
			fn(hookcore.ToolFailureEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				ToolName: ev.ToolName, Error: ev.ErrorMessage, WorkDir: ev.WorkDir, Raw: &ev,
			})
			return ToolFailureResult{}
		})
	}
}

// FileReadAdapter handles the unified pre-file-read hook for Cursor (beforeReadFile).
func FileReadAdapter(fn hookcore.FileReadFunc) (string, func()) {
	return "cursor-before-read-file", func() {
		hookcore.Run(func(ev ReadFilePreEvent) ReadFilePreResult {
			v := fn(hookcore.FileReadEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: ev.FilePath, Raw: &ev,
			})
			if v.Permit {
				return PermitRead()
			}
			return ForbidRead(v.Message)
		})
	}
}

package droid

import (
	"errors"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// This file owns the translation between Factory Droid's wire types and the
// unified hookcore vocabulary. The root agenthooks package wires these adapters
// into a registry; the per-platform logic lives here for locality.

// IdleAdapter handles the unified "agent finished" hook for Droid (Stop).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "droid-stop", func() {
		hookcore.Run(func(ev StopEvent) StopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentDroid, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if v.Proceed {
				return LetStop()
			}
			return HaltAndContinue(v.Feedback)
		})
	}
}

// SubagentIdleAdapter handles the unified "subagent finished" hook for Droid.
func SubagentIdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "droid-subagent-stop", func() {
		hookcore.Run(func(ev SubagentStopEvent) SubagentStopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentDroid, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if v.Proceed {
				return LetSubagentStop()
			}
			return HaltSubagent(v.Feedback)
		})
	}
}

// ToolAdapter handles the unified pre-tool-call hook for Droid.
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "droid-pre-tool-use", func() {
		hookcore.RunE(func(ev PreToolUseEvent) (PreToolUseResult, error) {
			kind, cmd := hookcore.StandardToolKind(ev.ToolName, ev.ToolInput)
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentDroid, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			if !v.Permit {
				// "Ask" is a JSON decision, not a hard block — emit it without exit 2.
				if v.NeedsConfirm {
					return AskUserAboutTool(v.Message), nil
				}
				return PreToolUseResult{}, errors.New(v.Message)
			}
			if v.RewrittenInput != nil {
				return ApproveToolUseWithInput(v.RewrittenInput), nil
			}
			return ApproveToolUse(), nil
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Droid (PostToolUse Write/Edit).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "droid-after-file-write", func() {
		hookcore.Run(func(ev PostToolUseEvent) PostToolUseResult {
			if !hookcore.IsStandardWriteTool(ev.ToolName) {
				return AcknowledgeToolUse()
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentDroid, SessionID: ev.SessionID,
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

// PromptAdapter handles the unified prompt-submit hook for Droid.
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "droid-user-prompt-submit", func() {
		hookcore.Run(func(ev UserPromptSubmitEvent) UserPromptSubmitResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentDroid, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
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

package droid

import "github.com/Checkmarx/ast-cx-hooks/internal/hookcore"

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

// preToolDecision maps a unified PreToolUse-style verdict to Droid's
// PreToolUseResult. Droid's PreToolUse output has no additionalContext or note
// channel, so a unified Context collapses (deny → plain deny, allow-with-context →
// plain allow) — unlike Droid's PostToolUse, which does carry additionalContext.
// It is the single seam both Droid PreToolUse gates use (ToolAdapter, FileEditAdapter).
func preToolDecision(v hookcore.ToolVerdict) PreToolUseResult {
	switch v.Path() {
	case hookcore.PathAllowWithInput:
		return ApproveToolUseWithInput(v.RewrittenInput)
	case hookcore.PathAsk:
		return AskUserAboutTool(v.Message)
	case hookcore.PathDeny, hookcore.PathDenyWithContext:
		return DenyToolUse(v.Message)
	default: // PathAllow / PathAllowWithContext / PathAllowWithNote
		return ApproveToolUse()
	}
}

// ToolAdapter handles the unified pre-tool-call hook for Droid.
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "droid-pre-tool-use", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			kind, cmd := hookcore.DroidTools.Kind(ev.ToolName, ev.ToolInput)
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentDroid, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileEditAdapter handles the unified pre-file-write GATE for Droid: PreToolUse
// scoped to the file-writing tools. Fires BEFORE the write and can DENY it; non-write
// tools are approved untouched so the generic droid-pre-tool-use gate owns them.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "droid-pre-file-write", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			if !hookcore.DroidTools.IsWrite(ev.ToolName) {
				return ApproveToolUse()
			}
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentDroid, SessionID: ev.SessionID,
				FilePath: hookcore.DroidTools.FilePath(ev.ToolInput),
				Changes:  hookcore.DroidTools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Droid (PostToolUse Write/Edit).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "droid-after-file-write", func() {
		hookcore.Run(func(ev PostToolUseEvent) PostToolUseResult {
			if !hookcore.DroidTools.IsWrite(ev.ToolName) {
				return AcknowledgeToolUse()
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentDroid, SessionID: ev.SessionID,
				FilePath: hookcore.DroidTools.FilePath(ev.ToolInput),
				Changes:  hookcore.DroidTools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			if v.Reject {
				// Droid PostToolDetails carries additionalContext, so a reject can also
				// attach context (e.g. a remediation instruction) alongside the block.
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

package gemini

import (
	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// This file owns the translation between Gemini's wire types and the unified
// hookcore vocabulary. The root agenthooks package wires these adapters into a
// registry; the per-platform logic lives here for locality.

// IdleAdapter handles the unified "agent finished" hook for Gemini (AfterAgent).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "gemini-after-agent", func() {
		hookcore.Run(func(ev AfterAgentEvent) AfterAgentResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentGemini, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if !v.Proceed {
				return RetryWithFeedback(v.Feedback)
			}
			return AcceptResponse()
		})
	}
}

// beforeToolDecision maps a unified PreToolUse-style verdict to Gemini's
// BeforeToolResult. Gemini's BeforeTool wire types carry only tool_input — no
// additionalContext and no ask channel — so remediation Context is folded into
// reason on deny (same workaround as Copilot CLI preToolUse #2585). An ask
// collapses to a deny (fail closed). Used by both BeforeTool gates: the tool-call
// gate (ToolAdapter) and the pre-file-write gate (FileEditAdapter).
func beforeToolDecision(v hookcore.ToolVerdict) BeforeToolResult {
	if !v.Permit {
		// Covers deny AND ask (no ask channel on BeforeTool): both block.
		if v.Context != "" {
			return DenyToolCallWithContext(v.Message, v.Context)
		}
		return DenyToolCall(v.Message)
	}
	if v.RewrittenInput != nil {
		return ApproveToolCallWithInput(v.RewrittenInput)
	}
	// Context unsupported on gemini BeforeTool allow (only tool_input rewrites).
	return ApproveToolCall()
}

// ToolAdapter handles the unified pre-tool-call hook for Gemini (BeforeTool).
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "gemini-before-tool", func() {
		hookcore.Run(func(ev BeforeToolEvent) BeforeToolResult {
			kind, cmd := hookcore.GeminiTools.Kind(ev.ToolName, ev.ToolInput)
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentGemini, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			return beforeToolDecision(v)
		})
	}
}

// FileEditAdapter handles the unified pre-file-write GATE for Gemini: BeforeTool
// scoped to the file-writing tools (write_file/replace). Fires BEFORE the write and
// can DENY it; non-write tools are approved untouched so the generic
// gemini-before-tool gate owns them. Changes are rebuilt from tool_input via
// FileChanges so ASCA can scan proposed content pre-write.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "gemini-before-file-tool", func() {
		hookcore.Run(func(ev BeforeToolEvent) BeforeToolResult {
			if !hookcore.GeminiTools.IsWrite(ev.ToolName) {
				return ApproveToolCall()
			}
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentGemini, SessionID: ev.SessionID,
				FilePath: hookcore.GeminiTools.FilePath(ev.ToolInput),
				Changes:  FileChanges(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir,
				Raw:      &ev,
			})
			return beforeToolDecision(v)
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Gemini (AfterTool write_file/replace).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "gemini-after-file-tool", func() {
		hookcore.Run(func(ev AfterToolEvent) AfterToolResult {
			if !hookcore.GeminiTools.IsWrite(ev.ToolName) {
				return AcknowledgeToolCall()
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentGemini, SessionID: ev.SessionID,
				FilePath: hookcore.GeminiTools.FilePath(ev.ToolInput), WorkDir: ev.WorkDir, Raw: &ev,
			})
			if v.Reject {
				// AfterTool is deliverable on gemini: AfterToolResult carries the block
				// (decision="deny" + reason) and AfterToolDetails.additionalContext, so a
				// reject-with-context is honored here (not fire-and-forget).
				if v.Context != "" {
					return DenyToolResultWithContext(v.Feedback, v.Context)
				}
				return DenyToolResult(v.Feedback)
			}
			if v.Footnote != "" {
				return AddToolAnnotation(v.Footnote)
			}
			return AcknowledgeToolCall()
		})
	}
}

// PromptAdapter handles the unified prompt-submit hook for Gemini (BeforeAgent).
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "gemini-before-agent", func() {
		hookcore.Run(func(ev BeforeAgentEvent) BeforeAgentResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentGemini, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
			})
			if !v.Accept {
				return RejectTurn(v.Message)
			}
			if v.Message != "" {
				return EnrichTurn(v.Message)
			}
			return AcceptTurn()
		})
	}
}

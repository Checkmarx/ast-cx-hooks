package codex

import "github.com/Checkmarx/ast-cx-hooks/internal/hookcore"

// This file owns the translation between Codex CLI's wire types and the
// unified hookcore vocabulary. The root agenthooks package wires these
// adapters into a registry; the per-platform logic lives here for locality.

// IdleAdapter handles the unified "agent finished" hook for Codex (Stop).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "codex-stop", func() {
		hookcore.Run(func(ev StopEvent) StopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCodex, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if v.Proceed {
				return LetStop()
			}
			return HaltAndContinue(v.Feedback)
		})
	}
}

// SubagentIdleAdapter handles the unified "subagent finished" hook for Codex.
func SubagentIdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "codex-subagent-stop", func() {
		hookcore.Run(func(ev SubagentStopEvent) SubagentStopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCodex, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if v.Proceed {
				return LetSubagentStop()
			}
			return HaltSubagent(v.Feedback)
		})
	}
}

// preToolDecision maps a unified PreToolUse-style verdict to Codex's
// PreToolUseResult via the shared Path() classifier. Codex's doc documents
// only "allow"/"deny" for PreToolUse output — no "ask" value, unlike Claude's
// allow/deny/ask/defer — so PathAsk collapses to a plain deny carrying the
// confirmation message, the same fail-toward-documented-behavior precedent
// gemini/adapters.go's beforeToolDecision uses when a platform has no ask
// channel. It is the single seam both PreToolUse gates use: the tool-call gate
// (ToolAdapter) and the pre-file-write gate (FileEditAdapter).
func preToolDecision(v hookcore.ToolVerdict) PreToolUseResult {
	switch v.Path() {
	case hookcore.PathAllowWithInput:
		return ApproveToolUseWithInput(v.RewrittenInput)
	case hookcore.PathAllowWithContext:
		return ApproveToolUseWithContext(v.Context)
	case hookcore.PathAllowWithNote:
		return ApproveToolUseWithNote(v.Message)
	case hookcore.PathAsk:
		// No ask/pending decision value is documented for Codex; collapse to deny.
		return DenyToolUse(v.Message)
	case hookcore.PathDenyWithContext:
		return DenyToolUseWithContext(v.Message, v.Context)
	case hookcore.PathDeny:
		return DenyToolUse(v.Message)
	default: // PathAllow
		return ApproveToolUse()
	}
}

// ToolAdapter handles the unified pre-tool-call hook for Codex (Bash + mcp__*).
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "codex-pre-tool-use", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			kind, cmd := hookcore.CodexTools.Kind(ev.ToolName, ev.ToolInput)
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCodex, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileEditAdapter handles the unified pre-file-write GATE for Codex: PreToolUse
// scoped to apply_patch. Fires BEFORE the write and can DENY it — the route a
// security scanner uses to block a vulnerable change before it reaches disk,
// optionally injecting remediation context on the deny. Non-write tools are
// approved untouched so the generic codex-pre-tool-use gate owns them.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "codex-pre-file-write", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			if !hookcore.CodexTools.IsWrite(ev.ToolName) {
				return ApproveToolUse()
			}
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentCodex, SessionID: ev.SessionID,
				FilePath: hookcore.CodexTools.FilePath(ev.ToolInput),
				Changes:  hookcore.CodexTools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Codex
// (PostToolUse scoped to apply_patch).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "codex-after-file-write", func() {
		hookcore.Run(func(ev PostToolUseEvent) PostToolUseResult {
			if !hookcore.CodexTools.IsWrite(ev.ToolName) {
				return AcknowledgeToolUse()
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentCodex, SessionID: ev.SessionID,
				FilePath: hookcore.CodexTools.FilePath(ev.ToolInput),
				Changes:  hookcore.CodexTools.Changes(ev.ToolName, ev.ToolInput),
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

// PromptAdapter handles the unified prompt-submit hook for Codex (UserPromptSubmit).
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "codex-user-prompt-submit", func() {
		hookcore.Run(func(ev UserPromptSubmitEvent) UserPromptSubmitResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentCodex, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
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

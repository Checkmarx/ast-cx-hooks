package claude

import (
	"fmt"
	"os"

	"github.com/Checkmarx/ast-cx-hooks/internal/codec"
	"github.com/Checkmarx/ast-cx-hooks/internal/hookcore"
)

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

// isPlainApprove reports whether out is a bare PreToolUse approve — "allow"
// with no note, no additionalContext, and no rewritten input.
func isPlainApprove(out PreToolUseResult) bool {
	return out.Details != nil && out.Details.Decision == "allow" &&
		out.Details.DecisionReason == "" && out.Details.ExtraContext == "" && out.Details.RewrittenInput == nil
}

// runPreToolUse is Claude's PreToolUse runner. Claude Code treats
// permissionDecision "allow" as "bypass the permission check entirely": it
// skips permission mode, settings allow/deny rules, and the developer's
// approval prompt. A clean scan is not consent, so a bare approve writes
// nothing to stdout and the host's normal permission flow decides. Deny, ask,
// additionalContext, and an allow that carries updatedInput are still
// JSON-encoded — a dropped deny is a vulnerability on disk, and a rewrite
// only applies when the grant travels with it. No os.Exit on the silent
// path, so in-process Dispatch tests stay alive.
func runPreToolUse(handler func(PreToolUseEvent) PreToolUseResult) {
	var in PreToolUseEvent
	if err := codec.DecodeStdin(&in); err != nil {
		fmt.Fprintf(os.Stderr, "agenthooks: stdin decode error: %v\n", err)
		os.Exit(0)
	}
	out := handler(in)
	if isPlainApprove(out) {
		return
	}
	if err := codec.EncodeStdout(out); err != nil {
		fmt.Fprintf(os.Stderr, "agenthooks: stdout encode error: %v\n", err)
		os.Exit(0)
	}
}

// preToolDecision maps a unified PreToolUse-style verdict to Claude's
// PreToolUseResult, honoring the full surface (allow/deny/ask + additionalContext
// + updatedInput) via the shared Path() classifier. It is the single seam both
// PreToolUse gates use: the tool-call gate (ToolAdapter) and the pre-file-write
// gate (FileEditAdapter). A plain allow is still returned as ApproveToolUse();
// runPreToolUse is what withholds it from stdout.
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
		runPreToolUse(func(ev PreToolUseEvent) PreToolUseResult {
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
// tools return a plain approve, which runPreToolUse withholds, so the generic
// claude-pre-tool-use gate owns them.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "claude-pre-file-write", func() {
		runPreToolUse(func(ev PreToolUseEvent) PreToolUseResult {
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

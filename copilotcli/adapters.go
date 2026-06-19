package copilotcli

import (
	"fmt"
	"os"
	"strings"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// This file owns the translation between Copilot CLI wire types and the unified
// hookcore vocabulary. The root agenthooks package wires these adapters into a
// registry; the per-platform logic lives here for locality.

// IdleAdapter handles the unified "agent finished" hook for Copilot CLI (agentStop).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "copilot-cli-stop", func() {
		hookcore.Run(func(ev StopEvent) StopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCopilotCLI, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if v.Proceed {
				return LetStop()
			}
			return HaltAndContinue(v.Feedback)
		})
	}
}

// SubagentIdleAdapter handles the unified "subagent finished" hook for Copilot CLI.
func SubagentIdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "copilot-cli-subagent-stop", func() {
		hookcore.Run(func(ev SubagentStopEvent) SubagentStopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCopilotCLI, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if v.Proceed {
				return LetSubagentStop()
			}
			return KeepSubagentRunning(v.Feedback)
		})
	}
}

// preToolDecision maps a unified PreToolUse-style verdict to Copilot CLI's FLAT
// preToolUse output. Remediation Context is delivered in the additionalContext
// field for forward-compatibility AND — because Copilot CLI does not yet honor
// additionalContext on preToolUse (github/copilot-cli#2585) — also folded into
// permissionDecisionReason on deny/ask, the field the CLI currently forwards to the
// agent. Used by both PreToolUse gates: the tool-call gate (ToolAdapter) and the
// pre-file-write gate (FileEditAdapter).
func preToolDecision(v hookcore.ToolVerdict) PreToolUseResult {
	if v.Permit {
		if v.RewrittenInput != nil {
			return ApproveToolUseWithInput(v.RewrittenInput)
		}
		if v.Context != "" {
			return ApproveToolUseWithContext(v.Message, v.Context)
		}
		if v.Message != "" {
			return ApproveToolUseWithNote(v.Message)
		}
		return ApproveToolUse()
	}
	if v.NeedsConfirm {
		if v.Context != "" {
			return AskUserAboutToolWithContext(v.Message, v.Context)
		}
		return AskUserAboutTool(v.Message)
	}
	if v.Context != "" {
		return DenyToolUseWithContext(v.Message, v.Context)
	}
	return DenyToolUse(v.Message)
}

// ToolAdapter handles the unified pre-tool-call hook for Copilot CLI (preToolUse).
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "copilot-cli-pre-tool-use", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			kind, cmd := hookcore.CopilotCLITools.Kind(ev.ToolName, ev.ToolInput)
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCopilotCLI, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileEditAdapter handles the unified pre-file-write GATE for Copilot CLI:
// preToolUse scoped to the file-writing tools (create/edit). Fires BEFORE the write
// and can DENY it; non-write tools are approved untouched so the generic
// copilot-cli-pre-tool-use gate owns them.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "copilot-cli-pre-file-write", func() {
		hookcore.Run(func(ev PreToolUseEvent) PreToolUseResult {
			if !hookcore.CopilotCLITools.IsWrite(ev.ToolName) {
				return ApproveToolUse()
			}
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentCopilotCLI, SessionID: ev.SessionID,
				FilePath: hookcore.CopilotCLITools.FilePath(ev.ToolInput),
				Changes:  hookcore.CopilotCLITools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			return preToolDecision(v)
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Copilot CLI
// (postToolUse on create/edit).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "copilot-cli-after-file-write", func() {
		hookcore.Run(func(ev PostToolUseEvent) PostToolUseResult {
			if !hookcore.CopilotCLITools.IsWrite(ev.ToolName) {
				return AcknowledgeToolUse()
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentCopilotCLI, SessionID: ev.SessionID,
				FilePath: hookcore.CopilotCLITools.FilePath(ev.ToolInput),
				Changes:  hookcore.CopilotCLITools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  ev.WorkDir, Raw: &ev,
			})
			if v.Reject {
				// postToolUse carries additionalContext, so fold the unified Context
				// (e.g. remediation steering) into the feedback delivered to the model.
				feedback := v.Feedback
				if v.Context != "" {
					feedback = strings.TrimSpace(v.Feedback + "\n" + v.Context)
				}
				return RejectToolResult(feedback)
			}
			if v.Footnote != "" {
				return AddToolContext(v.Footnote)
			}
			return AcknowledgeToolUse()
		})
	}
}

// ToolFailureAdapter handles the unified post-tool-failure hook for Copilot CLI
// (postToolUseFailure). The CLI cannot block here; feedback rides as additionalContext.
func ToolFailureAdapter(fn hookcore.ToolFailureFunc) (string, func()) {
	return "copilot-cli-post-tool-use-failure", func() {
		hookcore.Run(func(ev PostToolUseFailureEvent) PostToolUseFailureResult {
			v := fn(hookcore.ToolFailureEvent{
				Agent: hookcore.AgentCopilotCLI, SessionID: ev.SessionID, ToolName: ev.ToolName,
				Error: ev.Error, WorkDir: ev.WorkDir, Raw: &ev,
			})
			if v.Reject {
				return AnnotateFailure(v.Feedback)
			}
			if v.Footnote != "" {
				return AnnotateFailure(v.Footnote)
			}
			return AcknowledgeFailure()
		})
	}
}

// PromptAdapter handles the unified prompt-submit hook for Copilot CLI
// (userPromptSubmitted). This event is OBSERVATIONAL: the CLI does not process its
// output, so a Reject cannot block the prompt — it is logged and ignored.
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "copilot-cli-user-prompt-submit", func() {
		hookcore.Run(func(ev UserPromptSubmitEvent) UserPromptSubmitResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentCopilotCLI, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
			})
			if !v.Accept {
				fmt.Fprintln(os.Stderr, "agenthooks: copilot-cli user-prompt-submit output is not processed by Copilot CLI; reject ignored")
			}
			return AcknowledgePrompt()
		})
	}
}

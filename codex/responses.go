package codex

import (
	"encoding/json"

	"github.com/Checkmarx/ast-cx-hooks/internal/hookcore"
)

// --- Stop responses ---

// LetStop returns a decision that allows Codex to stop normally.
func LetStop() StopResult {
	return StopResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// HaltAndContinue blocks Codex from stopping and provides feedback to continue working.
func HaltAndContinue(reason string) StopResult {
	return StopResult{Decision: "block", Reason: reason}
}

// --- SubagentStop responses ---

// LetSubagentStop allows the subagent to stop normally.
func LetSubagentStop() SubagentStopResult {
	return SubagentStopResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// HaltSubagent blocks the subagent from stopping and provides feedback to continue working.
func HaltSubagent(reason string) SubagentStopResult {
	return SubagentStopResult{Decision: "block", Reason: reason}
}

// --- PreToolUse responses ---

// ApproveToolUse allows the tool call with no message.
func ApproveToolUse() PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{EventName: "PreToolUse", Decision: "allow"},
	}
}

// ApproveToolUseWithNote allows the tool call and surfaces a note to the user.
func ApproveToolUseWithNote(note string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "allow", DecisionReason: note,
		},
	}
}

// ApproveToolUseWithInput allows the tool call but rewrites its input before execution.
// updated is the full replacement tool_input as raw JSON.
func ApproveToolUseWithInput(updated json.RawMessage) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{EventName: "PreToolUse", Decision: "allow", RewrittenInput: updated},
	}
}

// ApproveToolUseWithContext allows the tool call and injects additional context for the agent.
func ApproveToolUseWithContext(ctx string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "allow", ExtraContext: ctx,
		},
	}
}

// DenyToolUse blocks the tool call and sends reason to the agent. Codex's doc
// documents no "ask" decision value, so this is also the response used to
// collapse an ask-style verdict — see adapters.go's preToolDecision.
func DenyToolUse(reason string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "deny", DecisionReason: reason,
		},
	}
}

// DenyToolUseWithContext blocks the tool call, sends reason to the agent, and
// injects additionalContext alongside the denial (e.g. an instruction to run a
// remediation skill).
func DenyToolUseWithContext(reason, ctx string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "deny", DecisionReason: reason, ExtraContext: ctx,
		},
	}
}

// --- PostToolUse responses ---

// AcknowledgeToolUse allows normal post-tool flow with no feedback.
func AcknowledgeToolUse() PostToolUseResult {
	return PostToolUseResult{}
}

// AddToolContext appends additional context for the agent after the tool completes.
func AddToolContext(ctx string) PostToolUseResult {
	return PostToolUseResult{
		Details: &PostToolDetails{EventName: "PostToolUse", ExtraContext: ctx},
	}
}

// RejectToolResult injects feedback into the agent after the tool completes.
func RejectToolResult(reason string) PostToolUseResult {
	return PostToolUseResult{Decision: "block", Reason: reason}
}

// RejectToolResultWithContext injects blocking feedback into the agent after the
// tool completes AND appends additionalContext (e.g. an instruction to run a
// remediation skill on the findings that caused the reject).
func RejectToolResultWithContext(reason, ctx string) PostToolUseResult {
	return PostToolUseResult{
		Decision: "block",
		Reason:   reason,
		Details:  &PostToolDetails{EventName: "PostToolUse", ExtraContext: ctx},
	}
}

// --- UserPromptSubmit responses ---

// ApprovePrompt allows the prompt to proceed.
func ApprovePrompt() UserPromptSubmitResult {
	return UserPromptSubmitResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// RejectPrompt blocks the prompt and shows reason to the user.
func RejectPrompt(reason string) UserPromptSubmitResult {
	return UserPromptSubmitResult{Decision: "block", Reason: reason}
}

// AppendToPrompt allows the prompt and injects additional context into the agent's system prompt.
func AppendToPrompt(ctx string) UserPromptSubmitResult {
	return UserPromptSubmitResult{
		ResultBase: ResultBase{Proceed: hookcore.Ptr(true)},
		Details:    &PromptSubmitDetails{EventName: "UserPromptSubmit", ExtraContext: ctx},
	}
}

// --- SessionStart responses (Extra) ---

// AcknowledgeSession allows the session to start with no additional context.
func AcknowledgeSession() SessionStartResult {
	return SessionStartResult{}
}

// InjectSessionContext injects additional context into the agent at session start.
func InjectSessionContext(ctx string) SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{EventName: "SessionStart", ExtraContext: ctx},
	}
}

// --- SubagentStart responses (Extra) ---

// AcknowledgeSubagentStart allows the subagent to start with no additional context.
func AcknowledgeSubagentStart() SubagentStartResult {
	return SubagentStartResult{}
}

// InjectSubagentContext injects additional context into the subagent at start.
func InjectSubagentContext(ctx string) SubagentStartResult {
	return SubagentStartResult{
		Details: &SubagentStartDetails{EventName: "SubagentStart", ExtraContext: ctx},
	}
}

// --- PermissionRequest responses (Extra) ---

// AllowPermission grants the requested permission.
func AllowPermission() PermissionRequestResult {
	return PermissionRequestResult{
		Details: &PermissionRequestDetails{
			EventName: "PermissionRequest",
			Decision:  &PermissionDecision{Behavior: "allow"},
		},
	}
}

// DenyPermission denies the requested permission, optionally with a message.
func DenyPermission(message string) PermissionRequestResult {
	return PermissionRequestResult{
		Details: &PermissionRequestDetails{
			EventName: "PermissionRequest",
			Decision:  &PermissionDecision{Behavior: "deny", Message: message},
		},
	}
}

// --- PreCompact / PostCompact responses (Extra) ---

// AcknowledgeCompact allows compaction to proceed with no changes.
func AcknowledgeCompact() PreCompactResult {
	return PreCompactResult{}
}

// AcknowledgePostCompact allows the session to continue after compaction.
func AcknowledgePostCompact() PostCompactResult {
	return PostCompactResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// --- SessionEnd responses (Extra) ---

// AcknowledgeSessionEnd acknowledges the session end (informational only).
func AcknowledgeSessionEnd() SessionEndResult {
	return SessionEndResult{}
}

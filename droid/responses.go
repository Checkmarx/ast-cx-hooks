package droid

import (
	"encoding/json"

	"github.com/Checkmarx/ast-cx-hooks/internal/hookcore"
)

// --- Stop responses ---

// LetStop allows Droid to stop normally.
func LetStop() StopResult {
	return StopResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// HaltAndContinue blocks Droid from stopping and provides feedback to continue.
func HaltAndContinue(reason string) StopResult {
	return StopResult{Decision: "block", Reason: reason}
}

// --- PreToolUse responses ---

// ApproveToolUse allows the tool call.
func ApproveToolUse() PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{EventName: "PreToolUse", Decision: "allow"},
	}
}

// ApproveToolUseWithInput allows the tool call but rewrites its input before execution.
func ApproveToolUseWithInput(updated json.RawMessage) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{EventName: "PreToolUse", Decision: "allow", RewrittenInput: updated},
	}
}

// DenyToolUse blocks the tool call and sends reason to Droid.
func DenyToolUse(reason string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "deny", DecisionReason: reason,
		},
	}
}

// AskUserAboutTool requests user confirmation before the tool call proceeds.
func AskUserAboutTool(reason string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "ask", DecisionReason: reason,
		},
	}
}

// --- PostToolUse responses ---

// AcknowledgeToolUse allows normal post-tool flow.
func AcknowledgeToolUse() PostToolUseResult {
	return PostToolUseResult{}
}

// AddToolContext appends additional context for Droid after the tool completes.
func AddToolContext(ctx string) PostToolUseResult {
	return PostToolUseResult{
		Details: &PostToolDetails{EventName: "PostToolUse", ExtraContext: ctx},
	}
}

// RejectToolResult injects feedback into Droid after the tool completes.
func RejectToolResult(reason string) PostToolUseResult {
	return PostToolUseResult{Decision: "block", Reason: reason}
}

// RejectToolResultWithContext injects feedback into Droid after the tool completes
// AND attaches additionalContext (e.g. an instruction to run a remediation skill on
// the findings that caused the reject) via the PostToolUse hookSpecificOutput.
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

// AppendToPrompt allows the prompt and injects additional context.
func AppendToPrompt(ctx string) UserPromptSubmitResult {
	return UserPromptSubmitResult{
		ResultBase: ResultBase{Proceed: hookcore.Ptr(true)},
		Details:    &PromptSubmitDetails{EventName: "UserPromptSubmit", ExtraContext: ctx},
	}
}

// --- SubagentStop responses ---

// LetSubagentStop allows a sub-droid task to stop normally.
func LetSubagentStop() SubagentStopResult {
	return SubagentStopResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// HaltSubagent blocks the subagent from stopping and provides feedback to continue.
func HaltSubagent(reason string) SubagentStopResult {
	return SubagentStopResult{Decision: "block", Reason: reason}
}

// --- Notification responses ---

// AcknowledgeNotification accepts a notification with no action.
func AcknowledgeNotification() NotificationResult { return NotificationResult{} }

// NotifyWithSystemMessage surfaces a system message to the user on a notification.
func NotifyWithSystemMessage(msg string) NotificationResult {
	return NotificationResult{ResultBase: ResultBase{SystemNote: msg}}
}

// --- SessionStart responses ---

// AcknowledgeSession allows the session to start.
func AcknowledgeSession() SessionStartResult { return SessionStartResult{} }

// InjectSessionContext injects context at session start.
func InjectSessionContext(ctx string) SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{EventName: "SessionStart", ExtraContext: ctx},
	}
}

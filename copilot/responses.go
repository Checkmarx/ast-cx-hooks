package copilot

import (
	"encoding/json"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// --- Stop responses ---

// LetStop returns a decision that allows Copilot to stop normally.
func LetStop() StopResult {
	return StopResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// HaltAndContinue blocks Copilot from stopping and provides feedback to continue working.
// VS Code Copilot reads the Stop decision from hookSpecificOutput, not top-level.
func HaltAndContinue(reason string) StopResult {
	return StopResult{Details: &StopDetails{EventName: "Stop", Decision: "block", Reason: reason}}
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
func ApproveToolUseWithInput(updated json.RawMessage) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{EventName: "PreToolUse", Decision: "allow", RewrittenInput: updated},
	}
}

// DenyToolUse blocks the tool call and sends reason to the agent.
func DenyToolUse(reason string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "deny", DecisionReason: reason,
		},
	}
}

// AskUserAboutTool asks the user to confirm before allowing the tool call.
func AskUserAboutTool(reason string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "ask", DecisionReason: reason,
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

// --- UserPromptSubmit responses ---

// ApprovePrompt allows the prompt to proceed.
func ApprovePrompt() UserPromptSubmitResult {
	return UserPromptSubmitResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// RejectPrompt blocks the prompt submission.
// VS Code Copilot's UserPromptSubmit supports only the common output format,
// so rejection uses continue=false + stopReason (not a decision/block field).
func RejectPrompt(reason string) UserPromptSubmitResult {
	return UserPromptSubmitResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(false), HaltReason: reason}}
}

// AppendToPrompt is a no-op on VS Code Copilot: UserPromptSubmit does not support
// additionalContext injection (common output format only). Retained for unified-API parity.
func AppendToPrompt(string) UserPromptSubmitResult {
	return UserPromptSubmitResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// --- SessionStart responses ---

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

// --- SubagentStart responses ---

// AcknowledgeSubagentStart lets a subagent spawn with no injected context.
func AcknowledgeSubagentStart() SubagentStartResult { return SubagentStartResult{} }

// InjectSubagentContext injects additional context into a spawned subagent's conversation.
func InjectSubagentContext(ctx string) SubagentStartResult {
	return SubagentStartResult{
		Details: &SubagentStartDetails{EventName: "SubagentStart", ExtraContext: ctx},
	}
}

// --- SubagentStop responses ---

// LetSubagentStop allows the subagent to stop normally.
func LetSubagentStop() SubagentStopResult {
	return SubagentStopResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// KeepSubagentRunning blocks subagent shutdown with feedback for it to continue.
func KeepSubagentRunning(reason string) SubagentStopResult {
	return SubagentStopResult{Decision: "block", Reason: reason}
}

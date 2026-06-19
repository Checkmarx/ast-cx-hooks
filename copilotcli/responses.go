package copilotcli

import "encoding/json"

// This file builds the FLAT response payloads the Copilot CLI expects. Unlike the
// VS Code Copilot extension, there is no hookSpecificOutput wrapper and no
// continue/stopReason envelope.

// --- agentStop (Stop) responses ---

// LetStop allows the agent to stop normally (empty output = default behavior).
func LetStop() StopResult { return StopResult{} }

// HaltAndContinue forces another agent turn using reason as the next prompt.
func HaltAndContinue(reason string) StopResult {
	return StopResult{Decision: "block", Reason: reason}
}

// --- subagentStop responses ---

// LetSubagentStop allows the subagent to stop normally.
func LetSubagentStop() SubagentStopResult { return SubagentStopResult{} }

// KeepSubagentRunning forces another subagent turn using reason as the next prompt.
func KeepSubagentRunning(reason string) SubagentStopResult {
	return SubagentStopResult{Decision: "block", Reason: reason}
}

// --- preToolUse responses ---

// ApproveToolUse allows the tool call (explicit allow).
func ApproveToolUse() PreToolUseResult {
	return PreToolUseResult{Decision: "allow"}
}

// ApproveToolUseWithNote allows the tool call and surfaces a reason to the agent.
func ApproveToolUseWithNote(note string) PreToolUseResult {
	return PreToolUseResult{Decision: "allow", DecisionReason: note}
}

// ApproveToolUseWithInput allows the tool call but substitutes its arguments
// (flat modifiedArgs, the CLI's name for an input override).
func ApproveToolUseWithInput(modified json.RawMessage) PreToolUseResult {
	return PreToolUseResult{Decision: "allow", ModifiedArgs: modified}
}

// DenyToolUse blocks the tool call and sends reason to the agent.
func DenyToolUse(reason string) PreToolUseResult {
	return PreToolUseResult{Decision: "deny", DecisionReason: reason}
}

// AskUserAboutTool asks the user to confirm the tool call. Under cloud agent this
// is treated as deny (no user is available), per the CLI docs.
func AskUserAboutTool(reason string) PreToolUseResult {
	return PreToolUseResult{Decision: "ask", DecisionReason: reason}
}

// ApproveToolUseWithContext allows the tool call and attaches remediation context
// via additionalContext. See PreToolUseResult.ExtraContext for the #2585 caveat
// (the field is currently ignored by Copilot CLI on preToolUse).
func ApproveToolUseWithContext(note, ctx string) PreToolUseResult {
	return PreToolUseResult{Decision: "allow", DecisionReason: note, ExtraContext: ctx}
}

// DenyToolUseWithContext blocks the tool call and delivers remediation context two
// ways: in additionalContext (forward-compat) and folded into permissionDecisionReason —
// the field Copilot CLI currently forwards to the agent while it ignores
// additionalContext on preToolUse (github/copilot-cli#2585).
func DenyToolUseWithContext(reason, ctx string) PreToolUseResult {
	return PreToolUseResult{
		Decision:       "deny",
		DecisionReason: mergeReasonContext(reason, ctx),
		ExtraContext:   ctx,
	}
}

// AskUserAboutToolWithContext mirrors DenyToolUseWithContext for the ask decision.
func AskUserAboutToolWithContext(reason, ctx string) PreToolUseResult {
	return PreToolUseResult{
		Decision:       "ask",
		DecisionReason: mergeReasonContext(reason, ctx),
		ExtraContext:   ctx,
	}
}

// mergeReasonContext joins a deny/ask reason with any remediation Context into a
// single permissionDecisionReason string, so the agent still receives the context
// while Copilot CLI ignores the dedicated additionalContext field (#2585).
func mergeReasonContext(reason, ctx string) string {
	if ctx == "" {
		return reason
	}
	if reason == "" {
		return ctx
	}
	return reason + "\n\n" + ctx
}

// --- postToolUse responses ---

// AcknowledgeToolUse keeps the original tool result unchanged.
func AcknowledgeToolUse() PostToolUseResult { return PostToolUseResult{} }

// AddToolContext appends additional guidance for the model after the tool result.
func AddToolContext(ctx string) PostToolUseResult {
	return PostToolUseResult{ExtraContext: ctx}
}

// ReplaceToolResult substitutes the tool's result text seen by the model.
func ReplaceToolResult(text string) PostToolUseResult {
	return PostToolUseResult{Modified: &ModifiedResult{ResultType: "success", TextResultForLLM: text}}
}

// RejectToolResult feeds corrective guidance back to the model via additionalContext.
// The CLI cannot block a completed tool, so a unified Reject surfaces as context.
func RejectToolResult(reason string) PostToolUseResult {
	return PostToolUseResult{ExtraContext: reason}
}

// --- postToolUseFailure responses ---

// AcknowledgeFailure takes no action on a tool failure.
func AcknowledgeFailure() PostToolUseFailureResult { return PostToolUseFailureResult{} }

// AnnotateFailure provides recovery guidance to the model after a tool failure.
func AnnotateFailure(ctx string) PostToolUseFailureResult {
	return PostToolUseFailureResult{ExtraContext: ctx}
}

// --- userPromptSubmitted responses ---

// AcknowledgePrompt is the only valid response: the CLI does not process output
// for userPromptSubmitted, so this hook is observational.
func AcknowledgePrompt() UserPromptSubmitResult { return UserPromptSubmitResult{} }

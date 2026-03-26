package claude

func boolPtr(b bool) *bool { return &b }

// --- Stop responses ---

// LetStop returns a decision that allows Claude to stop normally.
func LetStop() StopResult {
	return StopResult{ResultBase: ResultBase{Proceed: boolPtr(true)}}
}

// HaltAndContinue blocks Claude from stopping and provides feedback to continue working.
func HaltAndContinue(reason string) StopResult {
	return StopResult{Decision: "block", Reason: reason}
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
	return UserPromptSubmitResult{ResultBase: ResultBase{Proceed: boolPtr(true)}}
}

// RejectPrompt blocks the prompt and shows reason to the user.
func RejectPrompt(reason string) UserPromptSubmitResult {
	return UserPromptSubmitResult{Decision: "block", Reason: reason}
}

// AppendToPrompt allows the prompt and injects additional context into the agent's system prompt.
func AppendToPrompt(ctx string) UserPromptSubmitResult {
	return UserPromptSubmitResult{
		ResultBase: ResultBase{Proceed: boolPtr(true)},
		Details:    &PromptSubmitDetails{EventName: "UserPromptSubmit", ExtraContext: ctx},
	}
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

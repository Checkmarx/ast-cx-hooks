package droid

func boolPtr(b bool) *bool { return &b }

// --- Stop responses ---

// LetStop allows Droid to stop normally.
func LetStop() StopResult {
	return StopResult{ResultBase: ResultBase{Proceed: boolPtr(true)}}
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

// --- UserPromptSubmit responses ---

// ApprovePrompt allows the prompt to proceed.
func ApprovePrompt() UserPromptSubmitResult {
	return UserPromptSubmitResult{ResultBase: ResultBase{Proceed: boolPtr(true)}}
}

// RejectPrompt blocks the prompt and shows reason to the user.
func RejectPrompt(reason string) UserPromptSubmitResult {
	return UserPromptSubmitResult{Decision: "block", Reason: reason}
}

// AppendToPrompt allows the prompt and injects additional context.
func AppendToPrompt(ctx string) UserPromptSubmitResult {
	return UserPromptSubmitResult{
		ResultBase: ResultBase{Proceed: boolPtr(true)},
		Details:    &PromptSubmitDetails{EventName: "UserPromptSubmit", ExtraContext: ctx},
	}
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

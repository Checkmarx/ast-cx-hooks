package gemini

func boolPtr(b bool) *bool { return &b }

// --- BeforeTool responses ---

// ApproveToolCall allows the tool call to proceed.
func ApproveToolCall() BeforeToolResult {
	return BeforeToolResult{}
}

// DenyToolCall blocks the tool call; reason is fed back to the model.
func DenyToolCall(reason string) BeforeToolResult {
	return BeforeToolResult{ResultBase: ResultBase{Decision: "deny", Reason: reason}}
}

// --- AfterTool responses ---

// AcknowledgeToolCall accepts the tool result with no modification.
func AcknowledgeToolCall() AfterToolResult {
	return AfterToolResult{}
}

// AddToolAnnotation appends extra context to the tool result seen by the model.
func AddToolAnnotation(ctx string) AfterToolResult {
	return AfterToolResult{
		Details: &AfterToolDetails{ExtraContext: ctx},
	}
}

// --- BeforeAgent responses ---

// AcceptTurn allows the agent turn to proceed.
func AcceptTurn() BeforeAgentResult {
	return BeforeAgentResult{}
}

// RejectTurn blocks the agent turn; reason is shown to the user.
func RejectTurn(reason string) BeforeAgentResult {
	return BeforeAgentResult{
		ResultBase: ResultBase{Proceed: boolPtr(false), Reason: reason},
	}
}

// EnrichTurn allows the turn and appends additional context to the prompt.
func EnrichTurn(ctx string) BeforeAgentResult {
	return BeforeAgentResult{
		Details: &BeforeAgentDetails{ExtraContext: ctx},
	}
}

// --- AfterAgent responses ---

// AcceptResponse accepts the agent's response and ends the turn.
func AcceptResponse() AfterAgentResult {
	return AfterAgentResult{}
}

// RetryWithFeedback rejects the response; reason is fed back as retry context.
func RetryWithFeedback(reason string) AfterAgentResult {
	return AfterAgentResult{
		ResultBase: ResultBase{Proceed: boolPtr(false), Reason: reason},
	}
}

// --- SessionStart responses ---

// AcknowledgeSession allows the session to proceed.
func AcknowledgeSession() SessionStartResult { return SessionStartResult{} }

// InjectSessionContext injects context at session start.
func InjectSessionContext(ctx string) SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{ExtraContext: ctx},
	}
}

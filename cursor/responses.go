package cursor

func boolPtr(b bool) *bool { return &b }

// --- Permission helpers ---

// Permit allows the action with no message.
func Permit() PermissionResult {
	return PermissionResult{Permission: "allow"}
}

// PermitWithNote allows the action and surfaces a note in the Cursor UI.
func PermitWithNote(note string) PermissionResult {
	return PermissionResult{Permission: "allow", UserNote: note}
}

// Forbid denies the action and sends messages to the user and agent.
func Forbid(userMsg, agentMsg string) PermissionResult {
	return PermissionResult{Permission: "deny", UserNote: userMsg, AgentNote: agentMsg}
}

// RequestConfirmation asks the user to approve before the action proceeds.
func RequestConfirmation(userMsg, agentMsg string) PermissionResult {
	return PermissionResult{Permission: "ask", UserNote: userMsg, AgentNote: agentMsg}
}

// --- Stop helpers ---

// LetStop allows the agent loop to end naturally.
func LetStop() StopResult { return StopResult{} }

// SendFollowup prevents stopping by sending an automatic follow-up message.
func SendFollowup(message string) StopResult {
	return StopResult{FollowupText: message}
}

// --- Prompt helpers ---

// AcceptPrompt allows the prompt to be submitted.
func AcceptPrompt() PromptPreResult {
	return PromptPreResult{Continue: true}
}

// BlockPrompt prevents the prompt from being submitted and shows a message.
func BlockPrompt(note string) PromptPreResult {
	return PromptPreResult{Continue: false, UserNote: note}
}

// --- Session helpers ---

// AcknowledgeSession allows the session to start.
func AcknowledgeSession() SessionStartResult {
	return SessionStartResult{}
}

// InjectSessionEnv provides environment variables for the session.
func InjectSessionEnv(env map[string]string) SessionStartResult {
	return SessionStartResult{Env: env}
}

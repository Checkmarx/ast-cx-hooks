package cursor

import "encoding/json"

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

// --- preToolUse helpers ---

// PermitTool allows the tool to run with no message.
func PermitTool() ToolPreResult {
	return ToolPreResult{Permission: "allow"}
}

// ForbidTool denies the tool and sends messages to the user and agent.
func ForbidTool(userMsg, agentMsg string) ToolPreResult {
	return ToolPreResult{Permission: "deny", UserNote: userMsg, AgentNote: agentMsg}
}

// RewriteToolInput allows the tool to run with a rewritten input payload.
func RewriteToolInput(input json.RawMessage) ToolPreResult {
	return ToolPreResult{Permission: "allow", UpdatedInput: input}
}

// --- postToolUse helpers ---

// RewriteToolOutput replaces the tool output with the provided payload.
func RewriteToolOutput(output json.RawMessage) ToolPostResult {
	return ToolPostResult{UpdatedOutput: output}
}

// AddContext appends additional context for the agent after the tool runs.
func AddContext(context string) ToolPostResult {
	return ToolPostResult{ExtraContext: context}
}

// --- beforeReadFile helpers ---

// PermitRead allows the file to be read.
func PermitRead() ReadFilePreResult {
	return ReadFilePreResult{Permission: "allow"}
}

// ForbidRead denies the file read and shows a message in the UI.
func ForbidRead(userMsg string) ReadFilePreResult {
	return ReadFilePreResult{Permission: "deny", UserNote: userMsg}
}

// --- subagentStart helpers ---

// PermitSubagent allows the subagent to start.
func PermitSubagent() SubagentStartResult {
	return SubagentStartResult{Permission: "allow"}
}

// ForbidSubagent denies the subagent start and shows a message in the UI.
func ForbidSubagent(userMsg string) SubagentStartResult {
	return SubagentStartResult{Permission: "deny", UserNote: userMsg}
}

// --- subagentStop helpers ---

// LetSubagentStop allows the subagent loop to end naturally.
func LetSubagentStop() SubagentStopResult { return SubagentStopResult{} }

// SendSubagentFollowup prevents stopping by sending an automatic follow-up message.
func SendSubagentFollowup(message string) SubagentStopResult {
	return SubagentStopResult{FollowupText: message}
}

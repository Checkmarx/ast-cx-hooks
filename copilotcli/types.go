package copilotcli

import "encoding/json"

// EventBase contains fields present in every Copilot CLI hook payload, in the
// VS Code-compatible format (PascalCase event name in config -> snake_case fields).
type EventBase struct {
	EventName string `json:"hook_event_name"`
	SessionID string `json:"session_id"`
	// Timestamp is captured raw: Copilot CLI sends a numeric epoch (ms) while the
	// VS Code-compatible format sends an ISO 8601 string.
	Timestamp json.RawMessage `json:"timestamp"`
	WorkDir   string          `json:"cwd"`
}

// --- PreToolUse ---

// PreToolUseEvent is the payload for preToolUse hooks (before a tool executes).
type PreToolUseEvent struct {
	EventBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// UnmarshalJSON normalizes the two Copilot preToolUse payload shapes into one event:
//
//   - VS Code-compatible: {hook_event_name, session_id, timestamp(string), cwd,
//     tool_name, tool_input{...}}
//   - Copilot CLI native:  {sessionId, timestamp(number), cwd, toolName, toolArgs}
//     where toolArgs is a JSON-ENCODED STRING of the tool input object.
//
// Without this, the CLI's camelCase keys and string-encoded toolArgs left
// ToolName/ToolInput empty (so create/edit went unscanned), and its numeric
// timestamp broke the decode outright (the gate then failed open).
func (e *PreToolUseEvent) UnmarshalJSON(data []byte) error {
	var raw struct {
		EventName    string          `json:"hook_event_name"`
		SessionID    string          `json:"session_id"`
		SessionIDAlt string          `json:"sessionId"`
		Timestamp    json.RawMessage `json:"timestamp"`
		WorkDir      string          `json:"cwd"`
		ToolName     string          `json:"tool_name"`
		ToolNameAlt  string          `json:"toolName"`
		ToolInput    json.RawMessage `json:"tool_input"`
		ToolArgs     json.RawMessage `json:"toolArgs"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.EventName = raw.EventName
	e.SessionID = raw.SessionID
	if e.SessionID == "" {
		e.SessionID = raw.SessionIDAlt
	}
	e.Timestamp = raw.Timestamp
	e.WorkDir = raw.WorkDir
	e.ToolName = raw.ToolName
	if e.ToolName == "" {
		e.ToolName = raw.ToolNameAlt
	}
	e.ToolInput = normalizeToolInput(raw.ToolInput, raw.ToolArgs)
	return nil
}

// normalizeToolInput resolves the tool input object from either tool_input (an
// object, VS Code) or toolArgs (a JSON-encoded string, Copilot CLI). The CLI
// double-encodes toolArgs as a string, so it is unwrapped to the inner object
// before downstream parsing (file path, edit diff) sees it.
func normalizeToolInput(toolInput, toolArgs json.RawMessage) json.RawMessage {
	if len(toolInput) > 0 && string(toolInput) != "null" {
		return toolInput
	}
	if len(toolArgs) == 0 || string(toolArgs) == "null" {
		return toolArgs
	}
	var s string
	if json.Unmarshal(toolArgs, &s) == nil {
		return json.RawMessage(s) // toolArgs was a JSON-encoded string
	}
	return toolArgs // already an object
}

// PreToolUseResult is the FLAT JSON response for preToolUse hooks. Unlike the VS
// Code Copilot extension, the CLI uses top-level fields with no hookSpecificOutput
// wrapper, and the input override key is "modifiedArgs" (not "updatedInput").
type PreToolUseResult struct {
	Decision       string          `json:"permissionDecision,omitempty"`       // "allow", "deny", "ask"
	DecisionReason string          `json:"permissionDecisionReason,omitempty"` // required for deny
	ModifiedArgs   json.RawMessage `json:"modifiedArgs,omitempty"`             // substitute tool arguments
	// ExtraContext is emitted as additionalContext for the agent. Copilot CLI does
	// not yet honor this field on preToolUse (github/copilot-cli#2585); until it
	// does, deny/ask also fold the same text into DecisionReason so the agent still
	// receives it. Retained for forward-compatibility and parity with the other
	// (VS Code extension, Claude) adapters.
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- PostToolUse ---

// ToolResult mirrors the CLI's tool_result envelope (success path).
type ToolResult struct {
	ResultType       string `json:"result_type"`
	TextResultForLLM string `json:"text_result_for_llm"`
}

// PostToolUseEvent is the payload for postToolUse hooks (after a tool succeeds).
type PostToolUseEvent struct {
	EventBase
	ToolName   string          `json:"tool_name"`
	ToolInput  json.RawMessage `json:"tool_input"`
	ToolResult ToolResult      `json:"tool_result"`
}

// ModifiedResult is the replacement tool result for postToolUse (resultType must
// be "success"; a "failure" routes downstream to postToolUseFailure).
type ModifiedResult struct {
	ResultType       string `json:"resultType"`
	TextResultForLLM string `json:"textResultForLlm"`
}

// PostToolUseResult is the FLAT JSON response for postToolUse hooks.
type PostToolUseResult struct {
	Modified     *ModifiedResult `json:"modifiedResult,omitempty"`
	ExtraContext string          `json:"additionalContext,omitempty"`
}

// --- PostToolUseFailure ---

// PostToolUseFailureEvent is the payload for postToolUseFailure hooks.
type PostToolUseFailureEvent struct {
	EventBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	Error     string          `json:"error"`
}

// PostToolUseFailureResult is the FLAT JSON response for postToolUseFailure hooks.
// The CLI cannot block here — its only control surface is additionalContext.
type PostToolUseFailureResult struct {
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- agentStop (Stop) ---

// StopEvent is the payload for agentStop hooks (the main agent finishes a turn).
type StopEvent struct {
	EventBase
	TranscriptPath string `json:"transcript_path"`
	StopReason     string `json:"stop_reason"`
}

// StopResult is the FLAT JSON response for agentStop hooks. decision:"block"
// forces another turn using reason as the prompt.
type StopResult struct {
	Decision string `json:"decision,omitempty"` // "block" to force another turn
	Reason   string `json:"reason,omitempty"`
}

// --- subagentStop (SubagentStop) ---

// SubagentStopEvent is the payload for subagentStop hooks (a subagent completes).
type SubagentStopEvent struct {
	EventBase
	TranscriptPath   string `json:"transcript_path"`
	AgentName        string `json:"agent_name"`
	AgentDisplayName string `json:"agent_display_name,omitempty"`
	StopReason       string `json:"stop_reason"`
}

// SubagentStopResult is the FLAT JSON response for subagentStop hooks.
type SubagentStopResult struct {
	Decision string `json:"decision,omitempty"` // "block" to force another turn
	Reason   string `json:"reason,omitempty"`
}

// --- userPromptSubmitted (UserPromptSubmit) ---

// UserPromptSubmitEvent is the payload for userPromptSubmitted hooks.
type UserPromptSubmitEvent struct {
	EventBase
	Prompt string `json:"prompt"`
}

// UserPromptSubmitResult is the response for userPromptSubmitted hooks. The CLI
// does not process output for this event (it is observational), so the struct is
// intentionally empty.
type UserPromptSubmitResult struct{}

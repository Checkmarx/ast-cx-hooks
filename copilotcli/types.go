package copilotcli

import "encoding/json"

// EventBase contains fields present in every Copilot CLI hook payload, in the
// VS Code-compatible format (PascalCase event name in config -> snake_case fields).
type EventBase struct {
	EventName string `json:"hook_event_name"`
	SessionID string `json:"session_id"`
	Timestamp string `json:"timestamp"` // ISO 8601 in VS Code-compatible mode
	WorkDir   string `json:"cwd"`
}

// --- PreToolUse ---

// PreToolUseEvent is the payload for preToolUse hooks (before a tool executes).
type PreToolUseEvent struct {
	EventBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// PreToolUseResult is the FLAT JSON response for preToolUse hooks. Unlike the VS
// Code Copilot extension, the CLI uses top-level fields with no hookSpecificOutput
// wrapper, and the input override key is "modifiedArgs" (not "updatedInput").
type PreToolUseResult struct {
	Decision       string          `json:"permissionDecision,omitempty"`       // "allow", "deny", "ask"
	DecisionReason string          `json:"permissionDecisionReason,omitempty"` // required for deny
	ModifiedArgs   json.RawMessage `json:"modifiedArgs,omitempty"`             // substitute tool arguments
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

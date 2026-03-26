package gemini

import "encoding/json"

// EventBase contains fields present in every Gemini CLI hook payload.
type EventBase struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	WorkDir        string `json:"cwd"`
	EventName      string `json:"hook_event_name"`
	Timestamp      string `json:"timestamp"`
}

// ResultBase contains fields accepted by most Gemini CLI hook responses.
type ResultBase struct {
	Decision   string `json:"decision,omitempty"`   // "deny" to block
	Reason     string `json:"reason,omitempty"`
	Proceed    *bool  `json:"continue,omitempty"`
	HaltReason string `json:"stopReason,omitempty"`
	MuteOutput bool   `json:"suppressOutput,omitempty"`
	SystemNote string  `json:"systemMessage,omitempty"`
}

// --- BeforeTool ---

// MCPContext carries MCP server context for tool calls.
type MCPContext struct {
	ServerName string `json:"server_name,omitempty"`
	ServerURL  string `json:"server_url,omitempty"`
}

// BeforeToolEvent fires before a tool executes. Exit code 2 blocks it.
type BeforeToolEvent struct {
	EventBase
	ToolName    string          `json:"tool_name"`
	ToolInput   json.RawMessage `json:"tool_input"`
	MCPContext  *MCPContext     `json:"mcp_context,omitempty"`
	OriginalName string         `json:"original_request_name,omitempty"`
}

// BeforeToolResult is the JSON response for BeforeTool hooks.
type BeforeToolResult struct {
	ResultBase
	Details *BeforeToolDetails `json:"hookSpecificOutput,omitempty"`
}

// BeforeToolDetails carries BeforeTool-specific output fields.
type BeforeToolDetails struct {
	// RewrittenInput merges with the model's tool arguments when set.
	RewrittenInput json.RawMessage `json:"tool_input,omitempty"`
}

// --- AfterTool ---

// ToolResponse is the structured tool result provided in AfterTool events.
type ToolResponse struct {
	LLMContent    json.RawMessage `json:"llmContent,omitempty"`
	ReturnDisplay string          `json:"returnDisplay,omitempty"`
	Error         string          `json:"error,omitempty"`
}

// AfterToolEvent fires after a tool completes.
type AfterToolEvent struct {
	EventBase
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolResponse ToolResponse    `json:"tool_response"`
	MCPContext   *MCPContext     `json:"mcp_context,omitempty"`
	OriginalName string          `json:"original_request_name,omitempty"`
}

// AfterToolResult is the JSON response for AfterTool hooks.
type AfterToolResult struct {
	ResultBase
	Details *AfterToolDetails `json:"hookSpecificOutput,omitempty"`
}

// AfterToolDetails carries AfterTool-specific output fields.
type AfterToolDetails struct {
	// ExtraContext is appended to the tool result seen by the model.
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- BeforeAgent ---

// BeforeAgentEvent fires after a prompt is submitted, before planning begins.
// Exit code 2 aborts the turn and erases the prompt from context.
type BeforeAgentEvent struct {
	EventBase
	Prompt string `json:"prompt"`
}

// BeforeAgentResult is the JSON response for BeforeAgent hooks.
type BeforeAgentResult struct {
	ResultBase
	Details *BeforeAgentDetails `json:"hookSpecificOutput,omitempty"`
}

// BeforeAgentDetails carries BeforeAgent-specific output fields.
type BeforeAgentDetails struct {
	// ExtraContext is appended to the agent's prompt.
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- AfterAgent ---

// AfterAgentEvent fires when the agent loop completes.
// Exit code 2 rejects the response and triggers a retry with stderr as feedback.
type AfterAgentEvent struct {
	EventBase
	Prompt         string `json:"prompt"`
	PromptResponse string `json:"prompt_response"`
	HookActive     bool   `json:"stop_hook_active"` // true if triggered by a previous AfterAgent hook
}

// AfterAgentResult is the JSON response for AfterAgent hooks.
type AfterAgentResult struct {
	ResultBase
	Details *AfterAgentDetails `json:"hookSpecificOutput,omitempty"`
}

// AfterAgentDetails carries AfterAgent-specific output fields.
type AfterAgentDetails struct {
	// ClearContext clears the model's conversation memory when true.
	ClearContext bool `json:"clearContext,omitempty"`
}

// --- LLM request/response types ---

// LLMRequest represents a request to the Gemini model.
type LLMRequest struct {
	Model    string          `json:"model,omitempty"`
	Messages json.RawMessage `json:"messages,omitempty"`
	Config   json.RawMessage `json:"config,omitempty"`
}

// LLMResponse represents a response from the Gemini model.
type LLMResponse struct {
	Content json.RawMessage `json:"content,omitempty"`
}

// --- BeforeModel ---

// BeforeModelEvent fires before a request is sent to the LLM.
// Exit code 2 blocks the request and aborts the turn.
type BeforeModelEvent struct {
	EventBase
	LLMRequest LLMRequest `json:"llm_request"`
}

// BeforeModelResult is the JSON response for BeforeModel hooks.
type BeforeModelResult struct {
	ResultBase
	Details *BeforeModelDetails `json:"hookSpecificOutput,omitempty"`
}

// BeforeModelDetails carries BeforeModel-specific output fields.
type BeforeModelDetails struct {
	// OverrideRequest replaces the outgoing LLM request when set.
	OverrideRequest *LLMRequest `json:"llm_request,omitempty"`
	// SyntheticResponse provides a mock LLM response, skipping the actual LLM call.
	SyntheticResponse *LLMResponse `json:"llm_response,omitempty"`
}

// --- AfterModel ---

// AfterModelEvent fires after an LLM response chunk is received (fires per chunk during streaming).
type AfterModelEvent struct {
	EventBase
	LLMRequest  LLMRequest  `json:"llm_request"`
	LLMResponse LLMResponse `json:"llm_response"`
}

// AfterModelResult is the JSON response for AfterModel hooks.
type AfterModelResult struct {
	ResultBase
	Details *AfterModelDetails `json:"hookSpecificOutput,omitempty"`
}

// AfterModelDetails carries AfterModel-specific output fields.
type AfterModelDetails struct {
	// OverrideResponse replaces the received chunk when set.
	OverrideResponse *LLMResponse `json:"llm_response,omitempty"`
}

// --- BeforeToolSelection ---

// ToolConfig controls which tools the model may call.
type ToolConfig struct {
	Mode                string   `json:"mode,omitempty"`                 // "AUTO", "ANY", "NONE"
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"` // restrict to named tools
}

// BeforeToolSelectionEvent fires before the LLM selects a tool.
type BeforeToolSelectionEvent struct {
	EventBase
	LLMRequest LLMRequest `json:"llm_request"`
}

// BeforeToolSelectionResult is the JSON response for BeforeToolSelection hooks.
// Note: decision, continue, and systemMessage are not supported for this event.
type BeforeToolSelectionResult struct {
	Details *ToolSelectionDetails `json:"hookSpecificOutput,omitempty"`
}

// ToolSelectionDetails carries BeforeToolSelection-specific output.
type ToolSelectionDetails struct {
	ToolConfig *ToolConfig `json:"toolConfig,omitempty"`
}

// --- SessionStart ---

// SessionStartEvent is the payload for SessionStart hooks.
type SessionStartEvent struct {
	EventBase
	Trigger string `json:"source"` // "startup", "resume", "clear"
}

// SessionStartResult is the JSON response for SessionStart hooks.
type SessionStartResult struct {
	SystemNote string              `json:"systemMessage,omitempty"`
	Details    *SessionStartDetails `json:"hookSpecificOutput,omitempty"`
}

// SessionStartDetails carries session-start-specific output.
type SessionStartDetails struct {
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- SessionEnd ---

// SessionEndEvent is the payload for SessionEnd hooks.
type SessionEndEvent struct {
	EventBase
	Reason string `json:"reason"` // "exit", "clear", "logout", "prompt_input_exit", "other"
}

// SessionEndResult is the JSON response for SessionEnd hooks (advisory only).
type SessionEndResult struct {
	SystemNote string `json:"systemMessage,omitempty"`
}

// --- Notification ---

// NotificationEvent is the payload for Notification hooks.
type NotificationEvent struct {
	EventBase
	NotificationType string          `json:"notification_type"` // e.g. "ToolPermission"
	Message          string          `json:"message"`
	Details          json.RawMessage `json:"details,omitempty"`
}

// NotificationResult is the JSON response for Notification hooks (observational only).
type NotificationResult struct {
	SystemNote string `json:"systemMessage,omitempty"`
}

// --- PreCompress ---

// PreCompressEvent is the payload for PreCompress hooks.
type PreCompressEvent struct {
	EventBase
	Trigger string `json:"trigger"` // "auto" or "manual"
}

// PreCompressResult is the JSON response for PreCompress hooks (advisory only).
type PreCompressResult struct {
	SystemNote string `json:"systemMessage,omitempty"`
}

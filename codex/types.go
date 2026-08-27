package codex

import "encoding/json"

// EventBase contains fields present in every Codex CLI hook payload, per the
// documented input schema. Model and TurnID are Codex-specific additions on
// top of Claude Code's common-field set.
type EventBase struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	WorkDir        string `json:"cwd"`
	PermissionMode string `json:"permission_mode"` // "default", "acceptEdits", "plan", "dontAsk", "bypassPermissions"
	EventName      string `json:"hook_event_name"`
	Model          string `json:"model"`
	TurnID         string `json:"turn_id"`
}

// ResultBase contains fields accepted by most Codex hook responses.
type ResultBase struct {
	Proceed    *bool  `json:"continue,omitempty"`
	HaltReason string `json:"stopReason,omitempty"`
	MuteOutput bool   `json:"suppressOutput,omitempty"`
	SystemNote string `json:"systemMessage,omitempty"`
}

// --- Stop ---

// StopEvent is the payload sent to Stop hooks (fired when a turn completes).
type StopEvent struct {
	EventBase
}

// StopResult is the JSON response for Stop hooks.
type StopResult struct {
	ResultBase
	Decision string `json:"decision,omitempty"` // "block" to continue processing
	Reason   string `json:"reason,omitempty"`
}

// --- SubagentStop ---

// SubagentStopEvent is the payload for SubagentStop hooks (fired after a
// subagent completes). UNVERIFIED: the doc names the event but not its fields;
// these mirror Claude's SubagentStopEvent fields as a best-effort guess.
type SubagentStopEvent struct {
	EventBase
	AgentID             string `json:"agent_id"`
	AgentType           string `json:"agent_type"`
	AgentTranscriptPath string `json:"agent_transcript_path"`
}

// SubagentStopResult is the JSON response for SubagentStop hooks.
type SubagentStopResult struct {
	ResultBase
	Decision string `json:"decision,omitempty"` // "block" to continue processing
	Reason   string `json:"reason,omitempty"`
}

// --- PreToolUse ---

// PreToolUseEvent is the payload for PreToolUse hooks (fired before executing
// Bash, apply_patch, MCP tools, or local functions).
type PreToolUseEvent struct {
	EventBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	ToolUseID string          `json:"tool_use_id"`
}

// PreToolUseResult is the JSON response for PreToolUse hooks.
type PreToolUseResult struct {
	ResultBase
	Details *ToolPermission `json:"hookSpecificOutput,omitempty"`
}

// ToolPermission carries the permission decision for PreToolUse hooks.
// UNVERIFIED: the doc documents only "allow"/"deny" — no "ask" value, unlike
// Claude's allow/deny/ask/defer — so Decision is documented here as a
// two-value field; see adapters.go's preToolDecision for how an ask-style
// verdict is collapsed to deny.
type ToolPermission struct {
	EventName      string          `json:"hookEventName,omitempty"`
	Decision       string          `json:"permissionDecision,omitempty"`       // "allow" or "deny"
	DecisionReason string          `json:"permissionDecisionReason,omitempty"` // shown to the agent when denied
	RewrittenInput json.RawMessage `json:"updatedInput,omitempty"`             // optional input override
	ExtraContext   string          `json:"additionalContext,omitempty"`        // appended context for the agent
}

// --- PostToolUse ---

// PostToolUseEvent is the payload for PostToolUse hooks (fired after tools
// produce output).
type PostToolUseEvent struct {
	EventBase
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolResponse json.RawMessage `json:"tool_response"`
	ToolUseID    string          `json:"tool_use_id"`
}

// PostToolUseResult is the JSON response for PostToolUse hooks.
type PostToolUseResult struct {
	ResultBase
	Decision string           `json:"decision,omitempty"` // "block" to inject feedback
	Reason   string           `json:"reason,omitempty"`
	Details  *PostToolDetails `json:"hookSpecificOutput,omitempty"`
}

// PostToolDetails carries post-tool-use-specific output.
type PostToolDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- UserPromptSubmit ---

// UserPromptSubmitEvent is the payload for UserPromptSubmit hooks (fired
// before sending user input to the model).
type UserPromptSubmitEvent struct {
	EventBase
	Prompt string `json:"prompt"`
}

// UserPromptSubmitResult is the JSON response for UserPromptSubmit hooks.
type UserPromptSubmitResult struct {
	ResultBase
	Decision string               `json:"decision,omitempty"` // "block" to reject
	Reason   string               `json:"reason,omitempty"`
	Details  *PromptSubmitDetails `json:"hookSpecificOutput,omitempty"`
}

// PromptSubmitDetails carries prompt-submit-specific output.
type PromptSubmitDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- SessionStart (Extra) ---

// SessionStartEvent is the payload for SessionStart hooks (fired at startup,
// resume, clear, or compact). Extra: no unified-hook equivalent.
type SessionStartEvent struct {
	EventBase
	Trigger string `json:"source"` // "startup", "resume", "clear", "compact"
}

// SessionStartResult is the JSON response for SessionStart hooks.
type SessionStartResult struct {
	ResultBase
	Details *SessionStartDetails `json:"hookSpecificOutput,omitempty"`
}

// SessionStartDetails carries session-start-specific output.
type SessionStartDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- SessionEnd (Extra) ---

// SessionEndEvent is the payload for SessionEnd hooks (fired when a session
// closes or goes idle for 30 minutes). Extra: no unified-hook equivalent.
type SessionEndEvent struct {
	EventBase
}

// SessionEndResult is the JSON response for SessionEnd hooks (informational only).
type SessionEndResult struct {
	ResultBase
}

// --- SubagentStart (Extra) ---

// SubagentStartEvent is the payload for SubagentStart hooks (fired before a
// subagent launches). Extra: no unified-hook equivalent.
type SubagentStartEvent struct {
	EventBase
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`
}

// SubagentStartResult is the JSON response for SubagentStart hooks.
type SubagentStartResult struct {
	ResultBase
	Details *SubagentStartDetails `json:"hookSpecificOutput,omitempty"`
}

// SubagentStartDetails carries subagent-start-specific output.
type SubagentStartDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- PermissionRequest (Extra) ---

// PermissionRequestEvent is the payload for PermissionRequest hooks (fired
// before requesting approval, e.g. a shell escalation). Extra: no unified-hook
// equivalent (BeforeToolCall is backed by PreToolUse instead).
type PermissionRequestEvent struct {
	EventBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// PermissionRequestResult is the JSON response for PermissionRequest hooks.
type PermissionRequestResult struct {
	ResultBase
	Details *PermissionRequestDetails `json:"hookSpecificOutput,omitempty"`
}

// PermissionRequestDetails carries the nested permission decision.
type PermissionRequestDetails struct {
	EventName string              `json:"hookEventName,omitempty"`
	Decision  *PermissionDecision `json:"decision,omitempty"`
	Message   string              `json:"message,omitempty"`
}

// PermissionDecision is the nested allow/deny decision for PermissionRequest hooks.
type PermissionDecision struct {
	Behavior string `json:"behavior"` // "allow" or "deny"
	Message  string `json:"message,omitempty"`
}

// --- PreCompact (Extra) ---

// PreCompactEvent is the payload for PreCompact hooks (fired before chat
// compaction). Extra: no unified-hook equivalent.
type PreCompactEvent struct {
	EventBase
	Trigger string `json:"trigger"` // "manual" or "auto"
}

// PreCompactResult is the JSON response for PreCompact hooks.
type PreCompactResult struct {
	ResultBase
}

// --- PostCompact (Extra) ---

// PostCompactEvent is the payload for PostCompact hooks (fired after chat
// compaction). Extra: no unified-hook equivalent.
type PostCompactEvent struct {
	EventBase
	Trigger string `json:"trigger"` // "manual" or "auto"
}

// PostCompactResult is the JSON response for PostCompact hooks (informational only).
type PostCompactResult struct {
	ResultBase
}

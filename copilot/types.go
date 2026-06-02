package copilot

import "encoding/json"

// EventBase contains fields present in every VS Code Copilot agent hook payload.
//
// Copilot uses camelCase JSON keys (sessionId, hookEventName) while keeping
// transcript_path in snake_case to match the Claude Code reference protocol
// it was modeled on.
type EventBase struct {
	Timestamp      string `json:"timestamp"`
	WorkDir        string `json:"cwd"`
	SessionID      string `json:"sessionId"`
	EventName      string `json:"hookEventName"`
	TranscriptPath string `json:"transcript_path"`
}

// ResultBase contains fields accepted by every VS Code Copilot hook response.
type ResultBase struct {
	Proceed    *bool  `json:"continue,omitempty"`
	HaltReason string `json:"stopReason,omitempty"`
	SystemNote string `json:"systemMessage,omitempty"`
}

// --- Stop ---

// StopEvent is the payload sent to Stop hooks (agent finished responding).
type StopEvent struct {
	EventBase
	// HookActive is true when the current response was triggered by a previous Stop hook.
	HookActive bool `json:"stop_hook_active"`
}

// StopResult is the JSON response for Stop hooks.
//
// VS Code Copilot requires the Stop decision nested under hookSpecificOutput,
// unlike PostToolUse/SubagentStop which use a top-level decision/reason.
type StopResult struct {
	ResultBase
	Details *StopDetails `json:"hookSpecificOutput,omitempty"`
}

// StopDetails carries the Stop decision for VS Code Copilot.
type StopDetails struct {
	EventName string `json:"hookEventName,omitempty"`
	Decision  string `json:"decision,omitempty"` // "block" to prevent stopping
	Reason    string `json:"reason,omitempty"`
}

// --- SessionStart ---

// SessionStartEvent is the payload for SessionStart hooks.
type SessionStartEvent struct {
	EventBase
	Trigger string `json:"source"` // "startup", "resume", etc.
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

// --- UserPromptSubmit ---

// UserPromptSubmitEvent is the payload for UserPromptSubmit hooks.
type UserPromptSubmitEvent struct {
	EventBase
	Prompt string `json:"prompt"`
}

// UserPromptSubmitResult is the JSON response for UserPromptSubmit hooks.
//
// VS Code Copilot's UserPromptSubmit supports only the common output format
// (continue/stopReason/systemMessage); it does not honor a decision/block field
// or additionalContext injection.
type UserPromptSubmitResult struct {
	ResultBase
}

// --- PreToolUse ---

// PreToolUseEvent is the payload for PreToolUse hooks (fired before every tool call).
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
//
// Decision priority (most restrictive wins): deny > ask > allow.
type ToolPermission struct {
	EventName      string         `json:"hookEventName,omitempty"`
	Decision       string         `json:"permissionDecision,omitempty"`       // "allow", "deny", "ask"
	DecisionReason string         `json:"permissionDecisionReason,omitempty"` // shown to agent when denied
	RewrittenInput json.RawMessage `json:"updatedInput,omitempty"`             // optional input override
	ExtraContext   string         `json:"additionalContext,omitempty"`
}

// --- PostToolUse ---

// PostToolUseEvent is the payload for PostToolUse hooks (after a tool succeeds).
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

// --- PreCompact ---

// PreCompactEvent is the payload for PreCompact hooks (before context compaction).
type PreCompactEvent struct {
	EventBase
	Trigger string `json:"trigger"` // "manual" or "auto"
}

// PreCompactResult is the JSON response for PreCompact hooks (informational).
type PreCompactResult struct {
	ResultBase
}

// --- SubagentStart ---

// SubagentStartEvent is the payload for SubagentStart hooks (subagent spawned).
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

// SubagentStartDetails carries context injected into a spawned subagent.
type SubagentStartDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- SubagentStop ---

// SubagentStopEvent is the payload for SubagentStop hooks (subagent finished).
type SubagentStopEvent struct {
	EventBase
	AgentID    string `json:"agent_id"`
	AgentType  string `json:"agent_type"`
	HookActive bool   `json:"stop_hook_active"`
}

// SubagentStopResult is the JSON response for SubagentStop hooks.
type SubagentStopResult struct {
	ResultBase
	Decision string `json:"decision,omitempty"` // "block" to keep subagent running
	Reason   string `json:"reason,omitempty"`
}

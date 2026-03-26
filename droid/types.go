package droid

import "encoding/json"

// EventBase contains fields present in every Factory Droid hook payload.
type EventBase struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	WorkDir        string `json:"cwd"`
	PermissionMode string `json:"permission_mode"` // "off", "spec", "auto-low", "auto-medium", "auto-high"
	EventName      string `json:"hook_event_name"`
}

// ResultBase contains fields accepted by most Factory Droid hook responses.
type ResultBase struct {
	Proceed    *bool  `json:"continue,omitempty"`
	HaltReason string `json:"stopReason,omitempty"`
	MuteOutput bool   `json:"suppressOutput,omitempty"`
	SystemNote string  `json:"systemMessage,omitempty"`
}

// --- PreToolUse ---

// PreToolUseEvent is the payload for PreToolUse hooks (before every tool call).
// Exit code 2 blocks the tool and feeds stderr to Droid as feedback.
type PreToolUseEvent struct {
	EventBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// PreToolUseResult is the JSON response for PreToolUse hooks.
type PreToolUseResult struct {
	ResultBase
	Details *ToolPermission `json:"hookSpecificOutput,omitempty"`
}

// ToolPermission carries the permission decision for PreToolUse hooks.
type ToolPermission struct {
	EventName      string         `json:"hookEventName,omitempty"`
	Decision       string         `json:"permissionDecision,omitempty"`       // "allow", "deny", "ask"
	DecisionReason string         `json:"permissionDecisionReason,omitempty"`
	RewrittenInput map[string]any `json:"updatedInput,omitempty"`
}

// --- PostToolUse ---

// PostToolUseEvent is the payload for PostToolUse hooks (after a tool succeeds).
type PostToolUseEvent struct {
	EventBase
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolResponse json.RawMessage `json:"tool_response"`
}

// PostToolUseResult is the JSON response for PostToolUse hooks.
type PostToolUseResult struct {
	ResultBase
	Decision string           `json:"decision,omitempty"`
	Reason   string           `json:"reason,omitempty"`
	Details  *PostToolDetails `json:"hookSpecificOutput,omitempty"`
}

// PostToolDetails carries post-tool-use-specific output.
type PostToolDetails struct {
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
type UserPromptSubmitResult struct {
	ResultBase
	Decision string              `json:"decision,omitempty"` // "block" to reject
	Reason   string              `json:"reason,omitempty"`
	Details  *PromptSubmitDetails `json:"hookSpecificOutput,omitempty"`
}

// PromptSubmitDetails carries prompt-submit-specific output.
type PromptSubmitDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- Stop ---

// StopEvent is the payload for Stop hooks (fired when Droid finishes responding).
type StopEvent struct {
	EventBase
	HookActive bool `json:"stop_hook_active"`
}

// StopResult is the JSON response for Stop hooks.
type StopResult struct {
	ResultBase
	Decision string `json:"decision,omitempty"` // "block" to prevent stopping
	Reason   string `json:"reason,omitempty"`
}

// --- SubagentStop ---

// SubagentStopEvent is the payload for SubagentStop hooks (sub-droid task completes).
type SubagentStopEvent struct {
	EventBase
	HookActive bool `json:"stop_hook_active"`
}

// SubagentStopResult is the JSON response for SubagentStop hooks.
type SubagentStopResult struct {
	ResultBase
	Decision string `json:"decision,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// --- Notification ---

// NotificationEvent is the payload for Notification hooks.
type NotificationEvent struct {
	EventBase
	Message string `json:"message"`
}

// NotificationResult is the JSON response for Notification hooks.
type NotificationResult struct {
	ResultBase
}

// --- PreCompact ---

// PreCompactEvent is the payload for PreCompact hooks.
type PreCompactEvent struct {
	EventBase
	Trigger            string `json:"trigger"`             // "manual" or "auto"
	CustomInstructions string `json:"custom_instructions"`
}

// PreCompactResult is the JSON response for PreCompact hooks.
type PreCompactResult struct {
	ResultBase
}

// --- SessionStart ---

// SessionStartEvent is the payload for SessionStart hooks.
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

// --- SessionEnd ---

// SessionEndEvent is the payload for SessionEnd hooks.
type SessionEndEvent struct {
	EventBase
	Reason string `json:"reason"` // "clear", "logout", "prompt_input_exit", "other"
}

// SessionEndResult is the JSON response for SessionEnd hooks (informational).
type SessionEndResult struct {
	ResultBase
}

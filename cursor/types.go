package cursor

import "encoding/json"

// EventBase contains fields present in every Cursor hook payload.
type EventBase struct {
	ConversationID string   `json:"conversation_id"`
	GenerationID   string   `json:"generation_id"`
	Model          string   `json:"model"`
	EventName      string   `json:"hook_event_name"`
	CursorVersion  string   `json:"cursor_version"`
	WorkspaceRoots []string `json:"workspace_roots"`
	UserEmail      string   `json:"user_email"`
	TranscriptPath string   `json:"transcript_path"`
}

// PermissionResult is the standard output for hooks that gate an action.
type PermissionResult struct {
	Permission string `json:"permission"`            // "allow", "deny", or "ask"
	UserNote   string `json:"user_message,omitempty"` // shown in client UI
	AgentNote  string `json:"agent_message,omitempty"` // sent to the agent
}

// --- Shell execution ---

// ShellPreEvent is the payload for beforeShellExecution hooks.
type ShellPreEvent struct {
	EventBase
	Command string `json:"command"`
	WorkDir string `json:"cwd"`
	Timeout int    `json:"timeout"` // seconds
}

// ShellPreResult is the response type for beforeShellExecution hooks.
type ShellPreResult = PermissionResult

// ShellPostEvent is the payload for afterShellExecution hooks (fire-and-forget).
type ShellPostEvent struct {
	EventBase
	Command  string `json:"command"`
	Output   string `json:"output"`
	Duration int64  `json:"duration"` // milliseconds
}

// ShellPostResult is the response type for afterShellExecution hooks (unused by Cursor).
type ShellPostResult struct{}

// --- MCP execution ---

// MCPPreEvent is the payload for beforeMCPExecution hooks.
type MCPPreEvent struct {
	EventBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	ServerURL string          `json:"url,omitempty"`
	Command   string          `json:"command,omitempty"` // for command-based MCP servers
}

// MCPPreResult is the response type for beforeMCPExecution hooks.
type MCPPreResult = PermissionResult

// MCPPostEvent is the payload for afterMCPExecution hooks.
type MCPPostEvent struct {
	EventBase
	ToolName   string `json:"tool_name"`
	ToolInput  string `json:"tool_input"`  // JSON string
	ResultJSON string `json:"result_json"` // JSON string
	Duration   int64  `json:"duration"`    // milliseconds
}

// MCPPostResult is the response type for afterMCPExecution hooks (unused).
type MCPPostResult struct{}

// --- File operations ---

// FileEditEntry represents a single text replacement in a file edit.
type FileEditEntry struct {
	OldText string `json:"old_string"`
	NewText string `json:"new_string"`
}

// FileEditEvent is the payload for afterFileEdit hooks.
type FileEditEvent struct {
	EventBase
	FilePath string          `json:"file_path"`
	Edits    []FileEditEntry `json:"edits"`
}

// FileEditResult is the response type for afterFileEdit hooks (unused by Cursor).
type FileEditResult struct{}

// --- Prompt ---

// Attachment represents a file or context item attached to a prompt.
type Attachment struct {
	Type     string `json:"type"`
	FilePath string `json:"filePath"`
}

// PromptPreEvent is the payload for beforeSubmitPrompt hooks.
type PromptPreEvent struct {
	EventBase
	Prompt      string       `json:"prompt"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// PromptPreResult is the response type for beforeSubmitPrompt hooks.
type PromptPreResult struct {
	Continue bool   `json:"continue"`
	UserNote string `json:"user_message,omitempty"`
}

// --- Stop ---

// StopEvent is the payload for stop hooks (agent loop ends).
type StopEvent struct {
	EventBase
	Status    string `json:"status"`     // "completed", "aborted", "error"
	LoopCount int    `json:"loop_count"` // number of auto follow-ups so far (max 5)
}

// StopResult is the response type for stop hooks.
type StopResult struct {
	FollowupText string `json:"followup_message,omitempty"` // auto-submits a new message
}

// --- Session ---

// SessionStartEvent is the payload for sessionStart hooks.
type SessionStartEvent struct {
	EventBase
	SessionID         string `json:"session_id"`
	IsBackgroundAgent bool   `json:"is_background_agent"`
	ComposerMode      string `json:"composer_mode,omitempty"` // "agent", "ask", "edit"
}

// SessionStartResult is the response type for sessionStart hooks.
type SessionStartResult struct {
	Env           map[string]string `json:"env,omitempty"`
	ExtraContext   string            `json:"additional_context,omitempty"`
	Continue      *bool             `json:"continue,omitempty"`
	UserNote      string            `json:"user_message,omitempty"`
}

// SessionEndEvent is the payload for sessionEnd hooks.
type SessionEndEvent struct {
	EventBase
	SessionID         string `json:"session_id"`
	Reason            string `json:"reason"` // "completed", "aborted", "error", "window_close", "user_close"
	DurationMS        int64  `json:"duration_ms"`
	IsBackgroundAgent bool   `json:"is_background_agent"`
	FinalStatus       string `json:"final_status"`
	ErrorMessage      string `json:"error_message,omitempty"`
}

// SessionEndResult is the response type for sessionEnd hooks (fire-and-forget).
type SessionEndResult struct{}

// --- PreCompact ---

// PreCompactEvent is the payload for preCompact hooks.
type PreCompactEvent struct {
	EventBase
	Trigger            string `json:"trigger"` // "auto" or "manual"
	ContextUsagePct    int    `json:"context_usage_percent"`
	ContextTokens      int    `json:"context_tokens"`
	ContextWindowSize  int    `json:"context_window_size"`
	MessageCount       int    `json:"message_count"`
	MessagesToCompact  int    `json:"messages_to_compact"`
	IsFirstCompaction  bool   `json:"is_first_compaction"`
}

// PreCompactResult is the response type for preCompact hooks (observational only).
type PreCompactResult struct {
	UserNote string `json:"user_message,omitempty"`
}

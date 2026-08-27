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
	Permission string `json:"permission"`              // "allow", "deny", or "ask"
	UserNote   string `json:"user_message,omitempty"`  // shown in client UI
	AgentNote  string `json:"agent_message,omitempty"` // sent to the agent
}

// --- Shell execution ---

// ShellPreEvent is the payload for beforeShellExecution hooks.
type ShellPreEvent struct {
	EventBase
	Command string `json:"command"`
	WorkDir string `json:"cwd"`
	Sandbox bool   `json:"sandbox"` // command runs in a sandboxed environment
}

// ShellPreResult is the response type for beforeShellExecution hooks.
type ShellPreResult = PermissionResult

// ShellPostEvent is the payload for afterShellExecution hooks (fire-and-forget).
type ShellPostEvent struct {
	EventBase
	Command  string       `json:"command"`
	Output   string       `json:"output"`
	Duration Milliseconds `json:"duration"` // milliseconds
	Sandbox  bool         `json:"sandbox"`  // command ran in a sandboxed environment
}

// ShellPostResult is the response type for afterShellExecution hooks (unused by Cursor).
type ShellPostResult struct{}

// --- MCP execution ---

// MCPPreEvent is the payload for beforeMCPExecution hooks.
type MCPPreEvent struct {
	EventBase
	ToolName string `json:"tool_name"`
	// ToolInput is a JSON-encoded STRING of the tool params (per Cursor's
	// beforeMCPExecution wire format), not a nested JSON object — matching MCPPostEvent.
	ToolInput string `json:"tool_input"`
	ServerURL string `json:"url,omitempty"`
	Command   string `json:"command,omitempty"` // for command-based MCP servers
}

// MCPPreResult is the response type for beforeMCPExecution hooks.
type MCPPreResult = PermissionResult

// MCPPostEvent is the payload for afterMCPExecution hooks.
type MCPPostEvent struct {
	EventBase
	ToolName   string       `json:"tool_name"`
	ToolInput  string       `json:"tool_input"`  // JSON string
	ResultJSON string       `json:"result_json"` // JSON string
	Duration   Milliseconds `json:"duration"`    // milliseconds
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

// FileReadPreEvent is the payload for beforeReadFile hooks.
type FileReadPreEvent struct {
	EventBase
	FilePath string `json:"file_path"`
}

// FileReadPreResult is the response type for beforeReadFile hooks.
type FileReadPreResult = PermissionResult

// --- Prompt ---

// Attachment represents a file or context item attached to a prompt.
type Attachment struct {
	Type     string `json:"type"`      // "file" or "rule"
	FilePath string `json:"file_path"` // Cursor uses snake_case for attachment paths
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
//
// Current Cursor docs document only env and additional_context for sessionStart.
// Continue and UserNote are undocumented for this event (accepted by older schemas
// but not honored); kept for compatibility but should not be relied on.
type SessionStartResult struct {
	Env          map[string]string `json:"env,omitempty"`
	ExtraContext string            `json:"additional_context,omitempty"`
	Continue     *bool             `json:"continue,omitempty"`
	UserNote     string            `json:"user_message,omitempty"`
}

// SessionEndEvent is the payload for sessionEnd hooks.
type SessionEndEvent struct {
	EventBase
	SessionID         string       `json:"session_id"`
	Reason            string       `json:"reason"` // "completed", "aborted", "error", "window_close", "user_close"
	DurationMS        Milliseconds `json:"duration_ms"`
	IsBackgroundAgent bool         `json:"is_background_agent"`
	FinalStatus       string       `json:"final_status"`
	ErrorMessage      string       `json:"error_message,omitempty"`
}

// SessionEndResult is the response type for sessionEnd hooks (fire-and-forget).
type SessionEndResult struct{}

// --- PreCompact ---

// PreCompactEvent is the payload for preCompact hooks.
type PreCompactEvent struct {
	EventBase
	Trigger           string `json:"trigger"` // "auto" or "manual"
	ContextUsagePct   int    `json:"context_usage_percent"`
	ContextTokens     int    `json:"context_tokens"`
	ContextWindowSize int    `json:"context_window_size"`
	MessageCount      int    `json:"message_count"`
	MessagesToCompact int    `json:"messages_to_compact"`
	IsFirstCompaction bool   `json:"is_first_compaction"`
}

// PreCompactResult is the response type for preCompact hooks (observational only).
type PreCompactResult struct {
	UserNote string `json:"user_message,omitempty"`
}

// --- Tool use ---

// ToolPreEvent is the payload for preToolUse hooks (generic pre-tool gate).
type ToolPreEvent struct {
	EventBase
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolUseID    string          `json:"tool_use_id"`
	WorkDir      string          `json:"cwd"`
	ToolModel    string          `json:"model"`
	AgentMessage string          `json:"agent_message"`
}

// ToolPreResult is the response type for preToolUse hooks.
//
// It mirrors PermissionResult ("ask" is accepted but not enforced by Cursor)
// and adds UpdatedInput, which rewrites the tool input before execution.
type ToolPreResult struct {
	Permission string `json:"permission"`              // "allow", "deny", or "ask"
	UserNote   string `json:"user_message,omitempty"`  // shown in client UI
	AgentNote  string `json:"agent_message,omitempty"` // carried by the agent.v1 proto

	// Context is injected into the agent's context window on a deny. Cursor's CLI
	// preToolUse handler consumes only user_message and additional_context — it never
	// reads agent_message — so remediation instructions MUST be delivered here or the
	// agent never sees them. Set it alongside AgentNote: the proto carries both, and
	// different Cursor surfaces read different fields.
	Context string `json:"additional_context,omitempty"`

	UpdatedInput json.RawMessage `json:"updated_input,omitempty"` // rewritten tool input
}

// ToolPostEvent is the payload for postToolUse hooks.
type ToolPostEvent struct {
	EventBase
	ToolName   string          `json:"tool_name"`
	ToolInput  json.RawMessage `json:"tool_input"`
	ToolOutput string          `json:"tool_output"` // JSON-stringified tool output
	ToolUseID  string          `json:"tool_use_id"`
	WorkDir    string          `json:"cwd"`
	Duration   Milliseconds    `json:"duration"` // milliseconds
	ToolModel  string          `json:"model"`
}

// ToolPostResult is the response type for postToolUse hooks.
type ToolPostResult struct {
	UpdatedOutput json.RawMessage `json:"updated_mcp_tool_output,omitempty"` // rewritten tool output
	ExtraContext  string          `json:"additional_context,omitempty"`      // appended context for the agent
}

// ToolFailureEvent is the payload for postToolUseFailure hooks (observational, no output).
type ToolFailureEvent struct {
	EventBase
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolUseID    string          `json:"tool_use_id"`
	WorkDir      string          `json:"cwd"`
	ErrorMessage string          `json:"error_message"`
	FailureType  string          `json:"failure_type"` // "error", "timeout", "permission_denied"
	Duration     Milliseconds    `json:"duration"`     // milliseconds
	IsInterrupt  bool            `json:"is_interrupt"`
}

// ToolFailureResult is the response type for postToolUseFailure hooks (observational only).
type ToolFailureResult struct{}

// --- File read ---

// ReadFilePreEvent is the payload for beforeReadFile hooks (read-side gate).
type ReadFilePreEvent struct {
	EventBase
	FilePath    string       `json:"file_path"`
	Content     string       `json:"content"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// ReadFilePreResult is the response type for beforeReadFile hooks.
type ReadFilePreResult struct {
	Permission string `json:"permission"`             // "allow" or "deny"
	UserNote   string `json:"user_message,omitempty"` // shown in client UI
}

// --- Subagent ---

// SubagentStartEvent is the payload for subagentStart hooks.
type SubagentStartEvent struct {
	EventBase
	SubagentID           string `json:"subagent_id"`
	SubagentType         string `json:"subagent_type"`
	Task                 string `json:"task"`
	ParentConversationID string `json:"parent_conversation_id"`
	ToolCallID           string `json:"tool_call_id"`
	SubagentModel        string `json:"subagent_model"`
	IsParallelWorker     bool   `json:"is_parallel_worker"`
	GitBranch            string `json:"git_branch"`
}

// SubagentStartResult is the response type for subagentStart hooks.
type SubagentStartResult struct {
	Permission string `json:"permission"`             // "allow" or "deny"
	UserNote   string `json:"user_message,omitempty"` // shown in client UI
}

// SubagentStopEvent is the payload for subagentStop hooks.
type SubagentStopEvent struct {
	EventBase
	SubagentType        string       `json:"subagent_type"`
	Status              string       `json:"status"` // "completed", "error", "aborted"
	Task                string       `json:"task"`
	Description         string       `json:"description"`
	Summary             string       `json:"summary"`
	DurationMS          Milliseconds `json:"duration_ms"`
	MessageCount        int          `json:"message_count"`
	ToolCallCount       int          `json:"tool_call_count"`
	LoopCount           int          `json:"loop_count"`
	ModifiedFiles       []string     `json:"modified_files"`
	AgentTranscriptPath string       `json:"agent_transcript_path"`
}

// SubagentStopResult is the response type for subagentStop hooks (mirrors StopResult).
type SubagentStopResult struct {
	FollowupText string `json:"followup_message,omitempty"` // auto-submits a new message
}

// --- Agent response ---

// AgentResponseEvent is the payload for afterAgentResponse hooks (observational).
type AgentResponseEvent struct {
	EventBase
	Text string `json:"text"`
}

// AgentResponseResult is the response type for afterAgentResponse hooks (observational only).
type AgentResponseResult struct{}

// AgentThoughtEvent is the payload for afterAgentThought hooks (observational).
type AgentThoughtEvent struct {
	EventBase
	Text       string       `json:"text"`
	DurationMS Milliseconds `json:"duration_ms"`
}

// AgentThoughtResult is the response type for afterAgentThought hooks (observational only).
type AgentThoughtResult struct{}

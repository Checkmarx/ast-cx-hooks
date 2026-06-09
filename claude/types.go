package claude

import "encoding/json"

// EventBase contains fields present in every Claude Code hook payload.
type EventBase struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	WorkDir        string  `json:"cwd"`
	PermissionMode string  `json:"permission_mode"`
	EventName      string  `json:"hook_event_name"`
	Effort         *Effort `json:"effort,omitempty"`
}

// Effort carries the agent's configured effort level.
type Effort struct {
	Level string `json:"level"`
}

// ResultBase contains fields accepted by most Claude Code hook responses.
type ResultBase struct {
	Proceed    *bool  `json:"continue,omitempty"`
	HaltReason string `json:"stopReason,omitempty"`
	MuteOutput bool   `json:"suppressOutput,omitempty"`
	SystemNote string `json:"systemMessage,omitempty"`
	// TerminalSequence is an allow-listed terminal escape sequence (OSC 0/1/2/9/99/777
	// or BEL) Claude Code emits on the hook's behalf — e.g. a desktop notification or
	// window title. A universal output field available on every hook result.
	TerminalSequence string `json:"terminalSequence,omitempty"`
}

// --- Stop ---

// StopEvent is the payload sent to Stop hooks (fired when Claude finishes responding).
type StopEvent struct {
	EventBase
	// HookActive is true when the current response was triggered by a previous Stop hook,
	// allowing handlers to detect and break continuation loops.
	HookActive bool `json:"stop_hook_active"`
	// LastAssistantMessage is the text of Claude's final response, so handlers can
	// inspect it without parsing the transcript.
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
}

// StopResult is the JSON response for Stop hooks.
type StopResult struct {
	ResultBase
	Decision string `json:"decision,omitempty"` // "block" to prevent stopping
	Reason   string `json:"reason,omitempty"`
}

// --- SessionStart ---

// SessionStartEvent is the payload for SessionStart hooks.
type SessionStartEvent struct {
	EventBase
	Trigger string `json:"source"` // "startup", "resume", "clear", "compact"
	Model   string `json:"model"`
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
	// SessionTitle sets a display title for the session.
	SessionTitle string `json:"sessionTitle,omitempty"`
	// InitialUserMessage seeds an initial user message into the session.
	InitialUserMessage string `json:"initialUserMessage,omitempty"`
	// WatchPaths registers filesystem paths the session should watch for changes.
	WatchPaths []string `json:"watchPaths,omitempty"`
	// ReloadSkills requests the agent reload its skill set at session start.
	ReloadSkills bool `json:"reloadSkills,omitempty"`
}

// --- SessionEnd ---

// SessionEndEvent is the payload for SessionEnd hooks.
type SessionEndEvent struct {
	EventBase
	Reason string `json:"reason"` // "clear", "logout", "prompt_input_exit", "other"
}

// SessionEndResult is the JSON response for SessionEnd hooks (informational only).
type SessionEndResult struct {
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
type ToolPermission struct {
	EventName      string          `json:"hookEventName,omitempty"`
	Decision       string          `json:"permissionDecision,omitempty"`       // "allow", "deny", "ask", "defer"
	DecisionReason string          `json:"permissionDecisionReason,omitempty"` // shown to agent when denied
	RewrittenInput json.RawMessage `json:"updatedInput,omitempty"`             // optional input override
	ExtraContext   string          `json:"additionalContext,omitempty"`        // appended context for the agent
}

// --- PostToolUse ---

// PostToolUseEvent is the payload for PostToolUse hooks (fired after a tool succeeds).
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
	// UpdatedToolOutput replaces the tool result before the agent sees it
	// (distinct from ExtraContext, which only appends).
	UpdatedToolOutput json.RawMessage `json:"updatedToolOutput,omitempty"`
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
	Decision string               `json:"decision,omitempty"` // "block" to reject
	Reason   string               `json:"reason,omitempty"`
	Details  *PromptSubmitDetails `json:"hookSpecificOutput,omitempty"`
}

// PromptSubmitDetails carries prompt-submit-specific output.
type PromptSubmitDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
	// SessionTitle sets a display title for the session.
	SessionTitle string `json:"sessionTitle,omitempty"`
	// SuppressOriginalPrompt hides the user's original prompt from the transcript;
	// it applies when Decision == "block".
	SuppressOriginalPrompt bool `json:"suppressOriginalPrompt,omitempty"`
}

// --- Notification ---

// NotificationEvent is the payload for Notification hooks.
type NotificationEvent struct {
	EventBase
	Message          string `json:"message"`
	NotificationType string `json:"notification_type"`
}

// NotificationResult is the JSON response for Notification hooks.
type NotificationResult struct {
	ResultBase
}

// --- PreCompact ---

// PreCompactEvent is the payload for PreCompact hooks (fired before context compaction).
type PreCompactEvent struct {
	EventBase
	Trigger            string `json:"trigger"`             // "manual" or "auto"
	CustomInstructions string `json:"custom_instructions"` // existing compact instructions
}

// PreCompactResult is the JSON response for PreCompact hooks.
type PreCompactResult struct {
	ResultBase
}

// --- PostCompact ---

// PostCompactEvent is the payload for PostCompact hooks (fired after context compaction).
type PostCompactEvent struct {
	EventBase
	Trigger string `json:"trigger"` // "manual" or "auto"
}

// PostCompactResult is the JSON response for PostCompact hooks (informational only).
type PostCompactResult struct {
	ResultBase
}

// --- SubagentStart ---

// SubagentStartEvent is the payload for SubagentStart hooks (fired when a subagent begins).
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

// --- SubagentStop ---

// SubagentStopEvent is the payload for SubagentStop hooks (fired when a subagent finishes).
type SubagentStopEvent struct {
	EventBase
	AgentID             string `json:"agent_id"`
	AgentType           string `json:"agent_type"`
	AgentTranscriptPath string `json:"agent_transcript_path"`
	// HookActive is true when the subagent response was triggered by a previous SubagentStop hook,
	// allowing handlers to detect and break continuation loops.
	HookActive bool `json:"stop_hook_active"`
	// LastAssistantMessage is the subagent's final assistant message.
	LastAssistantMessage string `json:"last_assistant_message"`
}

// SubagentStopResult is the JSON response for SubagentStop hooks.
type SubagentStopResult struct {
	ResultBase
	Decision string `json:"decision,omitempty"` // "block" to prevent the subagent from stopping
	Reason   string `json:"reason,omitempty"`
}

// --- PostToolUseFailure ---

// PostToolUseFailureEvent is the payload for PostToolUseFailure hooks (fired after a tool errors).
type PostToolUseFailureEvent struct {
	EventBase
	ToolName    string          `json:"tool_name"`
	ToolInput   json.RawMessage `json:"tool_input"`
	Error       string          `json:"error"`
	ToolUseID   string          `json:"tool_use_id"`
	AgentID     string          `json:"agent_id"`
	AgentType   string          `json:"agent_type"`
	IsInterrupt bool            `json:"is_interrupt"`
	DurationMs  int64           `json:"duration_ms"`
}

// PostToolUseFailureResult is the JSON response for PostToolUseFailure hooks.
// Per the Claude Code hooks spec this hook cannot block; its only control is
// additionalContext (carried in Details).
type PostToolUseFailureResult struct {
	ResultBase
	Details *PostToolUseFailureDetails `json:"hookSpecificOutput,omitempty"`
}

// PostToolUseFailureDetails carries post-tool-failure-specific output.
type PostToolUseFailureDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- PermissionRequest ---

// PermissionRequestEvent is the payload for PermissionRequest hooks
// (fired when the agent requests permission to run a tool).
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
	Interrupt bool                `json:"interrupt,omitempty"`
}

// PermissionDecision is the nested allow/deny decision for PermissionRequest hooks.
type PermissionDecision struct {
	Behavior           string          `json:"behavior"` // "allow" or "deny"
	UpdatedInput       json.RawMessage `json:"updatedInput,omitempty"`
	UpdatedPermissions json.RawMessage `json:"updatedPermissions,omitempty"`
}

// --- Setup ---

// SetupEvent is the payload for Setup hooks (fired during environment setup).
type SetupEvent struct {
	EventBase
	Trigger string `json:"trigger"` // "init" or "maintenance"
}

// SetupResult is the JSON response for Setup hooks.
type SetupResult struct {
	ResultBase
	Details *SetupDetails `json:"hookSpecificOutput,omitempty"`
}

// SetupDetails carries setup-specific output.
type SetupDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- InstructionsLoaded ---

// InstructionsLoadedEvent is the payload for InstructionsLoaded hooks
// (fired when instruction/memory files are loaded). Observational only.
type InstructionsLoadedEvent struct {
	EventBase
	FilePath        string   `json:"file_path"`
	MemoryType      string   `json:"memory_type"` // "User", "Project", "Local", "Managed"
	LoadReason      string   `json:"load_reason"`
	Globs           []string `json:"globs"`
	TriggerFilePath string   `json:"trigger_file_path"`
	ParentFilePath  string   `json:"parent_file_path"`
}

// InstructionsLoadedResult is the JSON response for InstructionsLoaded hooks (informational only).
type InstructionsLoadedResult struct {
	ResultBase
}

// --- PermissionDenied ---

// PermissionDeniedEvent is the payload for PermissionDenied hooks
// (fired when a tool permission is denied).
type PermissionDeniedEvent struct {
	EventBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// PermissionDeniedResult is the JSON response for PermissionDenied hooks.
type PermissionDeniedResult struct {
	ResultBase
	Details *PermissionDeniedDetails `json:"hookSpecificOutput,omitempty"`
}

// PermissionDeniedDetails carries permission-denied-specific output.
type PermissionDeniedDetails struct {
	EventName string `json:"hookEventName,omitempty"`
	// Retry requests the agent re-attempt the denied action.
	Retry bool `json:"retry,omitempty"`
}

// --- ConfigChange ---

// ConfigChangeEvent is the payload for ConfigChange hooks (fired when configuration changes).
type ConfigChangeEvent struct {
	EventBase
	Source   string `json:"source"` // user_settings, project_settings, local_settings, policy_settings, skills
	FilePath string `json:"file_path"`
}

// ConfigChangeResult is the JSON response for ConfigChange hooks.
type ConfigChangeResult struct {
	ResultBase
	Decision string `json:"decision,omitempty"` // "block" to reject the change
	Reason   string `json:"reason,omitempty"`
}

// --- CwdChanged ---

// CwdChangedEvent is the payload for CwdChanged hooks (fired when the working directory changes).
// Observational only.
type CwdChangedEvent struct {
	EventBase
	OldCwd string `json:"old_cwd"`
	NewCwd string `json:"new_cwd"`
}

// CwdChangedResult is the JSON response for CwdChanged hooks (informational only).
type CwdChangedResult struct {
	ResultBase
}

// --- FileChanged ---

// FileChangedEvent is the payload for FileChanged hooks (fired when a watched file changes).
// Observational only.
type FileChangedEvent struct {
	EventBase
	FilePath string `json:"file_path"`
	Event    string `json:"event"` // "change", "add", "unlink"
}

// FileChangedResult is the JSON response for FileChanged hooks (informational only).
type FileChangedResult struct {
	ResultBase
}

// --- StopFailure ---

// StopFailureEvent is the payload for StopFailure hooks (fired when stopping fails).
// Observational only.
type StopFailureEvent struct {
	EventBase
	// Error is the error type, e.g. rate_limit, authentication_failed, billing_error,
	// invalid_request, model_not_found, server_error, max_output_tokens, unknown.
	Error                string `json:"error"`
	ErrorDetails         string `json:"error_details,omitempty"`
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
}

// StopFailureResult is the JSON response for StopFailure hooks (informational only).
type StopFailureResult struct {
	ResultBase
}

// --- UserPromptExpansion ---

// UserPromptExpansionEvent is the payload for UserPromptExpansion hooks
// (fired when a slash command or MCP prompt is expanded).
type UserPromptExpansionEvent struct {
	EventBase
	ExpansionType string `json:"expansion_type"` // "slash_command" or "mcp_prompt"
	CommandName   string `json:"command_name"`
	CommandArgs   string `json:"command_args"`
	CommandSource string `json:"command_source"`
	Prompt        string `json:"prompt"`
}

// UserPromptExpansionResult is the JSON response for UserPromptExpansion hooks.
type UserPromptExpansionResult struct {
	ResultBase
	Decision string                      `json:"decision,omitempty"` // "block" to reject the expansion
	Reason   string                      `json:"reason,omitempty"`
	Details  *UserPromptExpansionDetails `json:"hookSpecificOutput,omitempty"`
}

// UserPromptExpansionDetails carries prompt-expansion-specific output.
type UserPromptExpansionDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	ExtraContext string `json:"additionalContext,omitempty"`
}

// --- WorktreeCreate ---

// WorktreeCreateEvent is the payload for WorktreeCreate hooks (fired when a worktree is created).
type WorktreeCreateEvent struct {
	EventBase
	// Name is the slug for the new worktree (user-specified or auto-generated).
	Name string `json:"name"`
}

// WorktreeCreateResult is the JSON response for WorktreeCreate hooks.
type WorktreeCreateResult struct {
	ResultBase
	Details *WorktreeCreateDetails `json:"hookSpecificOutput,omitempty"`
}

// WorktreeCreateDetails carries worktree-create-specific output.
type WorktreeCreateDetails struct {
	EventName    string `json:"hookEventName,omitempty"`
	WorktreePath string `json:"worktreePath,omitempty"`
}

// --- WorktreeRemove ---

// WorktreeRemoveEvent is the payload for WorktreeRemove hooks (fired when a worktree is removed).
// Observational only.
type WorktreeRemoveEvent struct {
	EventBase
	WorktreePath string `json:"worktree_path"`
}

// WorktreeRemoveResult is the JSON response for WorktreeRemove hooks (informational only).
type WorktreeRemoveResult struct {
	ResultBase
}

// --- Elicitation ---

// ElicitationEvent is the payload for Elicitation hooks
// (fired when an MCP server requests structured input from the user).
type ElicitationEvent struct {
	EventBase
	ServerName      string          `json:"mcp_server_name"`
	Message         string          `json:"message,omitempty"`
	Mode            string          `json:"mode,omitempty"` // "form" or "url"
	RequestedSchema json.RawMessage `json:"requested_schema,omitempty"`
	URL             string          `json:"url,omitempty"` // present in url mode
	ElicitationID   string          `json:"elicitation_id,omitempty"`
}

// ElicitationResult is the JSON response for Elicitation hooks.
type ElicitationResult struct {
	ResultBase
	Details *ElicitationDetails `json:"hookSpecificOutput,omitempty"`
}

// ElicitationDetails carries elicitation-specific output.
type ElicitationDetails struct {
	EventName string          `json:"hookEventName,omitempty"`
	Action    string          `json:"action,omitempty"` // "accept", "decline", "cancel"
	Content   json.RawMessage `json:"content,omitempty"`
}

// --- ElicitationResult (post-elicitation) ---

// ElicitationResultEvent is the payload for ElicitationResult hooks
// (fired after the user responds to an elicitation form).
type ElicitationResultEvent struct {
	EventBase
	ServerName    string          `json:"mcp_server_name"`
	Action        string          `json:"action,omitempty"` // "accept", "decline", "cancel"
	Content       json.RawMessage `json:"content,omitempty"`
	Mode          string          `json:"mode,omitempty"`
	ElicitationID string          `json:"elicitation_id,omitempty"`
}

// ElicitationResultResult is the JSON response for ElicitationResult hooks.
// It reuses ElicitationDetails for its hookSpecificOutput shape.
type ElicitationResultResult struct {
	ResultBase
	Details *ElicitationDetails `json:"hookSpecificOutput,omitempty"`
}

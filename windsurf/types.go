package windsurf

import "encoding/json"

// EventBase contains fields present in every Windsurf Cascade hook payload.
type EventBase struct {
	ActionName   string `json:"agent_action_name"` // e.g. "pre_run_command"
	TrajectoryID string `json:"trajectory_id"`     // conversation identifier
	ExecutionID  string `json:"execution_id"`      // single agent turn identifier
	Timestamp    string `json:"timestamp"`          // ISO 8601
}

// --- pre_read_code / post_read_code ---

// ReadCodeInfo is the tool_info payload for read-code hooks.
type ReadCodeInfo struct {
	FilePath string `json:"file_path"`
}

// PreReadCodeEvent fires before Cascade reads a file. Exit code 2 blocks the read.
type PreReadCodeEvent struct {
	EventBase
	ToolInfo ReadCodeInfo `json:"tool_info"`
}

// PreReadCodeResult is the response for pre_read_code hooks (empty; blocking via exit code 2).
type PreReadCodeResult struct{}

// PostReadCodeEvent fires after Cascade successfully reads a file.
type PostReadCodeEvent struct {
	EventBase
	ToolInfo ReadCodeInfo `json:"tool_info"`
}

// PostReadCodeResult is the response for post_read_code hooks (informational).
type PostReadCodeResult struct{}

// --- pre_write_code / post_write_code ---

// CodeEdit represents a single text replacement in a write operation.
type CodeEdit struct {
	OldText string `json:"old_string"`
	NewText string `json:"new_string"`
}

// WriteCodeInfo is the tool_info payload for write-code hooks.
type WriteCodeInfo struct {
	FilePath string     `json:"file_path"`
	Edits    []CodeEdit `json:"edits"`
}

// PreWriteCodeEvent fires before Cascade writes or edits a file. Exit code 2 blocks.
type PreWriteCodeEvent struct {
	EventBase
	ToolInfo WriteCodeInfo `json:"tool_info"`
}

// PreWriteCodeResult is the response for pre_write_code hooks (blocking via exit code 2).
type PreWriteCodeResult struct{}

// PostWriteCodeEvent fires after Cascade successfully writes or edits a file.
type PostWriteCodeEvent struct {
	EventBase
	ToolInfo WriteCodeInfo `json:"tool_info"`
}

// PostWriteCodeResult is the response for post_write_code hooks (informational).
type PostWriteCodeResult struct{}

// --- pre_run_command / post_run_command ---

// RunCommandInfo is the tool_info payload for run-command hooks.
type RunCommandInfo struct {
	CommandLine string `json:"command_line"`
	WorkDir     string `json:"cwd"`
}

// PreRunCommandEvent fires before Cascade executes a terminal command. Exit code 2 blocks.
type PreRunCommandEvent struct {
	EventBase
	ToolInfo RunCommandInfo `json:"tool_info"`
}

// PreRunCommandResult is the response for pre_run_command hooks (blocking via exit code 2).
type PreRunCommandResult struct{}

// PostRunCommandEvent fires after a terminal command completes.
type PostRunCommandEvent struct {
	EventBase
	ToolInfo RunCommandInfo `json:"tool_info"`
}

// PostRunCommandResult is the response for post_run_command hooks (informational).
type PostRunCommandResult struct{}

// --- pre_mcp_tool_use / post_mcp_tool_use ---

// MCPToolUseInfo is the tool_info payload for MCP tool hooks.
type MCPToolUseInfo struct {
	ServerName string          `json:"mcp_server_name"`
	ToolName   string          `json:"mcp_tool_name"`
	Arguments  json.RawMessage `json:"mcp_tool_arguments"`
}

// PreMCPToolUseEvent fires before an MCP tool is invoked. Exit code 2 blocks.
type PreMCPToolUseEvent struct {
	EventBase
	ToolInfo MCPToolUseInfo `json:"tool_info"`
}

// PreMCPToolUseResult is the response for pre_mcp_tool_use hooks (blocking via exit code 2).
type PreMCPToolUseResult struct{}

// MCPPostToolUseInfo extends MCPToolUseInfo with the tool result.
type MCPPostToolUseInfo struct {
	MCPToolUseInfo
	Result string `json:"mcp_result"`
}

// PostMCPToolUseEvent fires after an MCP tool completes.
type PostMCPToolUseEvent struct {
	EventBase
	ToolInfo MCPPostToolUseInfo `json:"tool_info"`
}

// PostMCPToolUseResult is the response for post_mcp_tool_use hooks (informational).
type PostMCPToolUseResult struct{}

// --- pre_user_prompt ---

// UserPromptInfo is the tool_info payload for pre_user_prompt hooks.
type UserPromptInfo struct {
	UserPrompt string `json:"user_prompt"`
}

// PreUserPromptEvent fires before Cascade processes a user's prompt. Exit code 2 blocks.
type PreUserPromptEvent struct {
	EventBase
	ToolInfo UserPromptInfo `json:"tool_info"`
}

// PreUserPromptResult is the response for pre_user_prompt hooks (blocking via exit code 2).
type PreUserPromptResult struct{}

// --- post_cascade_response ---

// CascadeResponseInfo is the tool_info payload for post_cascade_response hooks.
type CascadeResponseInfo struct {
	Response string `json:"response"` // Markdown-formatted Cascade response
}

// PostCascadeResponseEvent fires asynchronously after Cascade responds. Non-blockable.
type PostCascadeResponseEvent struct {
	EventBase
	ToolInfo CascadeResponseInfo `json:"tool_info"`
}

// PostCascadeResponseResult is the response for post_cascade_response hooks (ignored).
type PostCascadeResponseResult struct{}

// --- post_cascade_response_with_transcript ---

// CascadeTranscriptInfo is the tool_info payload for post_cascade_response_with_transcript hooks.
type CascadeTranscriptInfo struct {
	TranscriptPath string `json:"transcript_path"` // path to JSONL transcript file
}

// PostCascadeTranscriptEvent fires asynchronously after Cascade responds with full transcript.
type PostCascadeTranscriptEvent struct {
	EventBase
	ToolInfo CascadeTranscriptInfo `json:"tool_info"`
}

// PostCascadeTranscriptResult is the response for post_cascade_response_with_transcript (ignored).
type PostCascadeTranscriptResult struct{}

// --- post_setup_worktree ---

// WorktreeInfo is the tool_info payload for post_setup_worktree hooks.
type WorktreeInfo struct {
	WorktreePath      string `json:"worktree_path"`
	RootWorkspacePath string `json:"root_workspace_path"`
}

// PostSetupWorktreeEvent fires after a git worktree is created.
type PostSetupWorktreeEvent struct {
	EventBase
	ToolInfo WorktreeInfo `json:"tool_info"`
}

// PostSetupWorktreeResult is the response for post_setup_worktree hooks (informational).
type PostSetupWorktreeResult struct{}

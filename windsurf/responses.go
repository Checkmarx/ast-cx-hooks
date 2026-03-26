package windsurf

// Pre-hooks in Windsurf block via exit code 2 (ProcessE returning an error),
// so response structs are empty. These constructors are provided for completeness
// and to document intent at call sites.

// AllowRead returns a response that permits the file read (exit 0).
func AllowRead() PreReadCodeResult { return PreReadCodeResult{} }

// AllowWrite returns a response that permits the file write (exit 0).
func AllowWrite() PreWriteCodeResult { return PreWriteCodeResult{} }

// AllowCommand returns a response that permits the terminal command (exit 0).
func AllowCommand() PreRunCommandResult { return PreRunCommandResult{} }

// AllowMCPTool returns a response that permits the MCP tool call (exit 0).
func AllowMCPTool() PreMCPToolUseResult { return PreMCPToolUseResult{} }

// AllowPrompt returns a response that permits the user prompt (exit 0).
func AllowPrompt() PreUserPromptResult { return PreUserPromptResult{} }

// AcknowledgeResponse returns a response for post_cascade_response (informational only).
func AcknowledgeResponse() PostCascadeResponseResult { return PostCascadeResponseResult{} }

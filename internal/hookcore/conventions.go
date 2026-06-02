package hookcore

import "encoding/json"

// StandardToolKind classifies a tool using the Bash + mcp__ naming convention
// shared by Claude, Droid, and Copilot, returning the kind and (for Bash) the command.
func StandardToolKind(toolName string, input json.RawMessage) (ToolKind, string) {
	if toolName == "Bash" {
		var v struct {
			Command string `json:"command"`
		}
		json.Unmarshal(input, &v) //nolint:errcheck // command may be absent; empty is fine
		return ToolKindShell, v.Command
	}
	if len(toolName) >= 5 && toolName[:5] == "mcp__" {
		return ToolKindMCP, ""
	}
	return ToolKindBuiltin, ""
}

// GeminiToolKind classifies a Gemini tool name (Gemini uses execute_bash /
// run_shell_command for shell and the mcp__ prefix for MCP tools).
func GeminiToolKind(toolName string) ToolKind {
	if len(toolName) >= 5 && toolName[:5] == "mcp__" {
		return ToolKindMCP
	}
	if toolName == "execute_bash" || toolName == "run_shell_command" {
		return ToolKindShell
	}
	return ToolKindBuiltin
}

// IsStandardWriteTool reports whether name is a file-writing tool under the
// Write/Edit/MultiEdit convention shared by Claude, Droid, and Copilot.
func IsStandardWriteTool(name string) bool {
	return name == "Write" || name == "Edit" || name == "MultiEdit"
}

// IsGeminiWriteTool reports whether name is a file-writing tool in Gemini's naming.
func IsGeminiWriteTool(name string) bool {
	return name == "write_file" || name == "replace_in_file" || name == "Write" || name == "Edit"
}

// StandardFilePath extracts the edited file path using the file_path convention
// shared by Claude, Droid, and Copilot.
func StandardFilePath(input json.RawMessage) string {
	var v struct {
		FilePath string `json:"file_path"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck // file_path may be absent; empty is fine
	return v.FilePath
}

// StandardWriteChanges extracts before/after diffs using the Write/Edit convention
// shared by Claude, Droid, and Copilot.
func StandardWriteChanges(toolName string, input json.RawMessage) []FileDiff {
	var v struct {
		Content   string `json:"content"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck // fields may be absent; empty is fine
	if toolName == "Edit" || toolName == "MultiEdit" {
		return []FileDiff{{Before: v.OldString, After: v.NewString}}
	}
	return []FileDiff{{Before: "", After: v.Content}}
}

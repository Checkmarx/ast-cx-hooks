package hookcore

import (
	"encoding/json"
	"strings"
)

// ToolConvention captures one agent's tool-naming scheme: how to recognize its
// shell and MCP tools, which tools write files, and where in tool_input the file
// path and edit text live. It replaces five near-duplicate helper families with a
// single deep type plus per-agent data instances, so the tool-naming knowledge for
// every supported agent lives in one place.
type ToolConvention struct {
	ShellTools   []string // tool names whose tool_input.command is a shell command
	MCPPrefix    string   // tool-name prefix marking an MCP tool ("" disables MCP matching)
	WriteTools   []string // tool names that create or modify files
	FilePathKeys []string // tool_input keys to try, in order, for the edited file path
	// Diff extracts before/after edit text; nil means this agent reports no diffs.
	Diff func(toolName string, input json.RawMessage) []FileDiff
}

// Kind classifies a tool name into shell / MCP / builtin, returning the shell
// command (from tool_input.command) when the tool is a shell tool.
func (c ToolConvention) Kind(toolName string, input json.RawMessage) (ToolKind, string) {
	if c.MCPPrefix != "" && strings.HasPrefix(toolName, c.MCPPrefix) {
		return ToolKindMCP, ""
	}
	for _, s := range c.ShellTools {
		if toolName == s {
			var v struct {
				Command string `json:"command"`
			}
			json.Unmarshal(input, &v) //nolint:errcheck // absent command is fine
			return ToolKindShell, v.Command
		}
	}
	return ToolKindBuiltin, ""
}

// IsWrite reports whether toolName is one of this agent's file-writing tools.
func (c ToolConvention) IsWrite(toolName string) bool {
	for _, w := range c.WriteTools {
		if toolName == w {
			return true
		}
	}
	return false
}

// FilePath returns the edited file path, trying each configured key in order.
func (c ToolConvention) FilePath(input json.RawMessage) string {
	var m map[string]json.RawMessage
	if json.Unmarshal(input, &m) != nil {
		return ""
	}
	for _, k := range c.FilePathKeys {
		if raw, ok := m[k]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil && s != "" {
				return s
			}
		}
	}
	return ""
}

// Changes returns the before/after edit text, or nil if this agent reports none.
func (c ToolConvention) Changes(toolName string, input json.RawMessage) []FileDiff {
	if c.Diff == nil {
		return nil
	}
	return c.Diff(toolName, input)
}

// standardDiff handles the Write/Edit/MultiEdit convention (Claude, Copilot):
// Edit/MultiEdit carry old_string/new_string; other write tools carry content.
func standardDiff(toolName string, input json.RawMessage) []FileDiff {
	var v struct {
		Content   string `json:"content"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	if toolName == "Edit" || toolName == "MultiEdit" {
		return []FileDiff{{Before: v.OldString, After: v.NewString}}
	}
	return []FileDiff{{Before: "", After: v.Content}}
}

// droidDiff handles Factory Droid's write tools: Edit carries old_string/new_string,
// ApplyPatch carries a unified patch (patch, else diff) surfaced best-effort as the
// After side, and Create carries content.
func droidDiff(toolName string, input json.RawMessage) []FileDiff {
	var v struct {
		Content   string `json:"content"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
		Patch     string `json:"patch"`
		Diff      string `json:"diff"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	switch toolName {
	case "Edit":
		return []FileDiff{{Before: v.OldString, After: v.NewString}}
	case "ApplyPatch":
		patch := v.Patch
		if patch == "" {
			patch = v.Diff
		}
		return []FileDiff{{Before: "", After: patch}}
	default: // Create
		return []FileDiff{{Before: "", After: v.Content}}
	}
}

// cliDiff handles the GitHub Copilot CLI write tools (best-effort, pending a live
// payload): edit carries old_string/new_string, create carries content.
func cliDiff(toolName string, input json.RawMessage) []FileDiff {
	var v struct {
		Content   string `json:"content"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	if toolName == "edit" {
		return []FileDiff{{Before: v.OldString, After: v.NewString}}
	}
	return []FileDiff{{Before: "", After: v.Content}}
}

var (
	// ClaudeTools is the Claude Code tool-naming convention (Bash + mcp__ + Write/Edit/MultiEdit).
	ClaudeTools = ToolConvention{
		ShellTools: []string{"Bash"}, MCPPrefix: "mcp__",
		WriteTools:   []string{"Write", "Edit", "MultiEdit"},
		FilePathKeys: []string{"file_path"}, Diff: standardDiff,
	}
	// DroidTools is the Factory Droid convention (Execute + mcp__ + Create/Edit/ApplyPatch).
	DroidTools = ToolConvention{
		ShellTools: []string{"Execute"}, MCPPrefix: "mcp__",
		WriteTools:   []string{"Create", "Edit", "ApplyPatch"},
		FilePathKeys: []string{"file_path"}, Diff: droidDiff,
	}
	// GeminiTools is the Gemini CLI convention (run_shell_command + single-underscore
	// mcp_ + write_file/replace). Gemini reports no before/after diffs.
	GeminiTools = ToolConvention{
		ShellTools: []string{"run_shell_command"}, MCPPrefix: "mcp_",
		WriteTools:   []string{"write_file", "replace"},
		FilePathKeys: []string{"file_path", "path"}, Diff: nil,
	}
	// CopilotTools is the VS Code Copilot convention (runTerminalCommand + mcp_ +
	// createFile/editFiles, camelCase filePath).
	CopilotTools = ToolConvention{
		ShellTools: []string{"runTerminalCommand"}, MCPPrefix: "mcp_",
		WriteTools:   []string{"createFile", "editFiles"},
		FilePathKeys: []string{"filePath"}, Diff: standardDiff,
	}
	// CopilotCLITools is the GitHub Copilot CLI convention (lowercase bash/powershell,
	// no MCP prefix documented, create/edit).
	CopilotCLITools = ToolConvention{
		ShellTools: []string{"bash", "powershell"}, MCPPrefix: "",
		WriteTools:   []string{"create", "edit"},
		FilePathKeys: []string{"file_path"}, Diff: cliDiff,
	}
)

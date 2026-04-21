package lifecycle

import (
	"encoding/json"
	"errors"

	"github.com/CheckmarxDev/ast-cx-hooks/claude"
	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
	"github.com/CheckmarxDev/ast-cx-hooks/droid"
	"github.com/CheckmarxDev/ast-cx-hooks/gemini"
	"github.com/CheckmarxDev/ast-cx-hooks/internal/dispatch"
	"github.com/CheckmarxDev/ast-cx-hooks/windsurf"
)

// ToolCallEvent provides a unified view of pre-execution events across all platforms.
type ToolCallEvent struct {
	Agent AgentID
	Kind  ToolKind

	// Command is the shell command string (shell executions only).
	Command string

	// WorkDir is the working directory for the execution.
	WorkDir string

	// ToolName is the tool identifier (MCP and builtin tools).
	// For MCP tools this follows the pattern mcp__<server>__<tool>.
	ToolName string

	// ToolArgs is the raw JSON arguments passed to the tool.
	ToolArgs json.RawMessage

	// ServerURL is the MCP server URL or name (Cursor/Windsurf/Gemini).
	ServerURL string

	// Raw holds the original platform-specific input.
	Raw any
}

// IsMCP returns true when this event represents an MCP tool call.
func (e ToolCallEvent) IsMCP() bool { return e.Kind == ToolKindMCP }

// IsShell returns true when this event represents a shell command execution.
func (e ToolCallEvent) IsShell() bool { return e.Kind == ToolKindShell }

// ToolVerdict is the decision returned by a BeforeToolCall handler.
type ToolVerdict struct {
	Permit       bool
	Message      string // reason shown to user (allow) or agent (deny)
	NeedsConfirm bool   // true = ask user for confirmation before proceeding
}

// Allow permits the tool call with no message.
func Allow() ToolVerdict { return ToolVerdict{Permit: true} }

// AllowWithNote permits the tool call and surfaces a note.
func AllowWithNote(msg string) ToolVerdict { return ToolVerdict{Permit: true, Message: msg} }

// Deny blocks the tool call and sends a reason to the agent.
func Deny(reason string) ToolVerdict { return ToolVerdict{Permit: false, Message: reason} }

// AskUser blocks pending user confirmation and explains why.
func AskUser(reason string) ToolVerdict {
	return ToolVerdict{Permit: false, NeedsConfirm: true, Message: reason}
}

// ToolCallFunc is the handler signature for BeforeToolCall.
type ToolCallFunc func(ToolCallEvent) ToolVerdict

// BeforeToolCall registers a unified handler for pre-execution events on all platforms:
//   - Claude Code   → "claude-pre-tool-use"      (Bash + mcp__* tools)
//   - Cursor        → "cursor-before-shell"       (shell)
//   - Cursor        → "cursor-before-mcp"         (MCP)
//   - Windsurf      → "windsurf-pre-run-command"  (shell, blocking via exit 2)
//   - Windsurf      → "windsurf-pre-mcp-tool-use" (MCP, blocking via exit 2)
//   - Factory Droid → "droid-pre-tool-use"        (Bash + mcp__* tools, blocking via exit 2)
//   - Gemini CLI    → "gemini-before-tool"        (all tools, blocking via exit 2)
func BeforeToolCall(fn ToolCallFunc) {
	// Claude Code — covers Bash and mcp__* tools via PreToolUse
	dispatch.AddRoute("claude-pre-tool-use", func() {
		dispatch.Process(func(ev claude.PreToolUseEvent) claude.PreToolUseResult {
			kind, cmd := claudeToolKind(ev.ToolName, ev.ToolInput)
			verdict := fn(ToolCallEvent{
				Agent: AgentClaude, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			if verdict.Permit {
				if verdict.Message != "" {
					return claude.ApproveToolUseWithNote(verdict.Message)
				}
				return claude.ApproveToolUse()
			}
			if verdict.NeedsConfirm {
				return claude.AskUserAboutTool(verdict.Message)
			}
			return claude.DenyToolUse(verdict.Message)
		})
	})

	// Cursor — shell
	dispatch.AddRoute("cursor-before-shell", func() {
		dispatch.Process(func(ev cursor.ShellPreEvent) cursor.ShellPreResult {
			verdict := fn(ToolCallEvent{
				Agent: AgentCursor, Kind: ToolKindShell,
				Command: ev.Command, WorkDir: ev.WorkDir, Raw: &ev,
			})
			return cursorPermissionResult(verdict)
		})
	})

	// Cursor — MCP
	dispatch.AddRoute("cursor-before-mcp", func() {
		dispatch.Process(func(ev cursor.MCPPreEvent) cursor.MCPPreResult {
			verdict := fn(ToolCallEvent{
				Agent: AgentCursor, Kind: ToolKindMCP,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput,
				ServerURL: ev.ServerURL, Command: ev.Command, Raw: &ev,
			})
			return cursorPermissionResult(verdict)
		})
	})

	// Windsurf — shell (blocking via exit code 2)
	dispatch.AddRoute("windsurf-pre-run-command", func() {
		dispatch.ProcessE(func(ev windsurf.PreRunCommandEvent) (windsurf.PreRunCommandResult, error) {
			verdict := fn(ToolCallEvent{
				Agent: AgentWindsurf, Kind: ToolKindShell,
				Command: ev.ToolInfo.CommandLine, WorkDir: ev.ToolInfo.WorkDir, Raw: &ev,
			})
			if !verdict.Permit {
				return windsurf.PreRunCommandResult{}, errors.New(verdict.Message)
			}
			return windsurf.AllowCommand(), nil
		})
	})

	// Windsurf — MCP (blocking via exit code 2)
	dispatch.AddRoute("windsurf-pre-mcp-tool-use", func() {
		dispatch.ProcessE(func(ev windsurf.PreMCPToolUseEvent) (windsurf.PreMCPToolUseResult, error) {
			verdict := fn(ToolCallEvent{
				Agent: AgentWindsurf, Kind: ToolKindMCP,
				ToolName: ev.ToolInfo.ToolName, ToolArgs: ev.ToolInfo.Arguments,
				ServerURL: ev.ToolInfo.ServerName, Raw: &ev,
			})
			if !verdict.Permit {
				return windsurf.PreMCPToolUseResult{}, errors.New(verdict.Message)
			}
			return windsurf.AllowMCPTool(), nil
		})
	})

	// Factory Droid — Bash and mcp__* (blocking via exit code 2).
	// Droid uses the same tool naming convention as Claude Code.
	dispatch.AddRoute("droid-pre-tool-use", func() {
		dispatch.ProcessE(func(ev droid.PreToolUseEvent) (droid.PreToolUseResult, error) {
			kind, cmd := claudeToolKind(ev.ToolName, ev.ToolInput) // same convention as Claude
			verdict := fn(ToolCallEvent{
				Agent: AgentDroid, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			if !verdict.Permit {
				return droid.PreToolUseResult{}, errors.New(verdict.Message)
			}
			if verdict.Message != "" {
				return droid.ApproveToolUseWithNote(verdict.Message), nil
			}
			return droid.ApproveToolUse(), nil
		})
	})

	// Gemini CLI — all tools (JSON deny response)
	dispatch.AddRoute("gemini-before-tool", func() {
		dispatch.Process(func(ev gemini.BeforeToolEvent) gemini.BeforeToolResult {
			kind := geminiToolKind(ev.ToolName)
			verdict := fn(ToolCallEvent{
				Agent: AgentGemini, Kind: kind,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput,
				Raw: &ev,
			})
			if !verdict.Permit {
				return gemini.DenyToolCall(verdict.Message)
			}
			return gemini.ApproveToolCall()
		})
	})
}

// claudeToolKind classifies a Claude Code / Factory Droid tool name into ToolKind
// and extracts the shell command string when the tool is Bash.
func claudeToolKind(toolName string, input json.RawMessage) (ToolKind, string) {
	if toolName == "Bash" {
		var v struct {
			Command string `json:"command"`
		}
		json.Unmarshal(input, &v) //nolint:errcheck
		return ToolKindShell, v.Command
	}
	if len(toolName) >= 5 && toolName[:5] == "mcp__" {
		return ToolKindMCP, ""
	}
	return ToolKindBuiltin, ""
}

// geminiToolKind classifies a Gemini CLI tool name into ToolKind.
func geminiToolKind(toolName string) ToolKind {
	if len(toolName) >= 5 && toolName[:5] == "mcp__" {
		return ToolKindMCP
	}
	if toolName == "execute_bash" || toolName == "run_shell_command" {
		return ToolKindShell
	}
	return ToolKindBuiltin
}

// cursorPermissionResult converts a ToolVerdict to a Cursor permission result.
func cursorPermissionResult(v ToolVerdict) cursor.PermissionResult {
	if v.Permit {
		if v.Message != "" {
			return cursor.PermitWithNote(v.Message)
		}
		return cursor.Permit()
	}
	if v.NeedsConfirm {
		return cursor.RequestConfirmation(v.Message, v.Message)
	}
	return cursor.Forbid(v.Message, v.Message)
}

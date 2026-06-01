package agenthooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/CheckmarxDev/ast-cx-hooks/claude"
	"github.com/CheckmarxDev/ast-cx-hooks/copilot"
	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
	"github.com/CheckmarxDev/ast-cx-hooks/droid"
	"github.com/CheckmarxDev/ast-cx-hooks/gemini"
	"github.com/CheckmarxDev/ast-cx-hooks/windsurf"
)

// AgentID identifies which AI coding agent triggered the hook.
type AgentID string

const (
	AgentClaude   AgentID = "claude"
	AgentCursor   AgentID = "cursor"
	AgentWindsurf AgentID = "windsurf"
	AgentDroid    AgentID = "droid"
	AgentGemini   AgentID = "gemini"
	AgentCopilot  AgentID = "copilot"
)

// =============================================================================
// WhenAgentIdle — unified Stop / AfterAgent / post_cascade_response handler
// =============================================================================

// AgentIdleEvent provides a unified view of "agent finished responding" events
// from all five platforms.
type AgentIdleEvent struct {
	Agent AgentID

	// SessionID is the conversation or session identifier.
	// Source: session_id (Claude/Droid), conversation_id (Cursor), trajectory_id (Windsurf), session_id (Gemini).
	SessionID string

	// WorkDir is the working directory at the time the hook fired (Claude/Droid only).
	WorkDir string

	// IsRepeat is true when this idle hook was itself triggered by a prior continuation.
	// Use this together with IsLooping() to break infinite loops.
	// Source: stop_hook_active (Claude/Droid/Gemini).
	IsRepeat bool

	// CompletionStatus is the agent's final status string (Cursor only).
	// Values: "completed", "aborted", "error".
	CompletionStatus string

	// AutoRetryCount is the number of auto follow-ups already triggered this session (Cursor only).
	// Cursor enforces a maximum of 5.
	AutoRetryCount int

	// Raw holds the original platform-specific input for advanced use.
	Raw any
}

// IsLooping returns true when continuing would create an infinite loop.
// For Claude, Droid, and Gemini it checks IsRepeat; for Cursor it checks AutoRetryCount.
func (e AgentIdleEvent) IsLooping() bool {
	switch e.Agent {
	case AgentClaude, AgentDroid, AgentGemini, AgentCopilot:
		return e.IsRepeat
	case AgentCursor:
		return e.AutoRetryCount >= 3
	default:
		return false
	}
}

// IdleVerdict is the decision returned by a WhenAgentIdle handler.
type IdleVerdict struct {
	// Proceed true = let the agent stop; false = continue working.
	Proceed  bool
	Feedback string // shown to the agent when Proceed is false
}

// Resume allows the agent to stop normally.
func Resume() IdleVerdict { return IdleVerdict{Proceed: true} }

// Interrupt prevents the agent from stopping and sends feedback for the next iteration.
func Interrupt(feedback string) IdleVerdict { return IdleVerdict{Proceed: false, Feedback: feedback} }

// AgentIdleFunc is the handler signature for WhenAgentIdle.
type AgentIdleFunc func(AgentIdleEvent) IdleVerdict

// WhenAgentIdle registers a unified handler for "agent finished responding" events
// on all five platforms. A single handler covers:
//   - Claude Code   → "claude-stop"
//   - Cursor        → "cursor-stop"
//   - Windsurf      → "windsurf-post-cascade-response" (fire-and-forget; Interrupt is logged but ignored)
//   - Factory Droid → "droid-stop"
//   - Gemini CLI    → "gemini-after-agent"
//   - VS Code Copilot → "copilot-stop"
func WhenAgentIdle(fn AgentIdleFunc) {
	AddRoute("claude-stop", func() {
		Process(func(ev claude.StopEvent) claude.StopResult {
			verdict := fn(AgentIdleEvent{
				Agent: AgentClaude, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if verdict.Proceed {
				return claude.LetStop()
			}
			return claude.HaltAndContinue(verdict.Feedback)
		})
	})

	AddRoute("cursor-stop", func() {
		Process(func(ev cursor.StopEvent) cursor.StopResult {
			verdict := fn(AgentIdleEvent{
				Agent: AgentCursor, SessionID: ev.ConversationID,
				CompletionStatus: ev.Status, AutoRetryCount: ev.LoopCount, Raw: &ev,
			})
			if verdict.Proceed {
				return cursor.LetStop()
			}
			return cursor.SendFollowup(verdict.Feedback)
		})
	})

	AddRoute("windsurf-post-cascade-response", func() {
		Process(func(ev windsurf.PostCascadeResponseEvent) windsurf.PostCascadeResponseResult {
			verdict := fn(AgentIdleEvent{
				Agent: AgentWindsurf, SessionID: ev.TrajectoryID, Raw: &ev,
			})
			if !verdict.Proceed {
				fmt.Fprintf(os.Stderr,
					"agenthooks: windsurf post-cascade-response is fire-and-forget; Interrupt(%q) ignored\n",
					verdict.Feedback)
			}
			return windsurf.AcknowledgeResponse()
		})
	})

	AddRoute("droid-stop", func() {
		Process(func(ev droid.StopEvent) droid.StopResult {
			verdict := fn(AgentIdleEvent{
				Agent: AgentDroid, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if verdict.Proceed {
				return droid.LetStop()
			}
			return droid.HaltAndContinue(verdict.Feedback)
		})
	})

	AddRoute("gemini-after-agent", func() {
		Process(func(ev gemini.AfterAgentEvent) gemini.AfterAgentResult {
			verdict := fn(AgentIdleEvent{
				Agent: AgentGemini, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if !verdict.Proceed {
				return gemini.RetryWithFeedback(verdict.Feedback)
			}
			return gemini.AcceptResponse()
		})
	})

	AddRoute("copilot-stop", func() {
		Process(func(ev copilot.StopEvent) copilot.StopResult {
			verdict := fn(AgentIdleEvent{
				Agent: AgentCopilot, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if verdict.Proceed {
				return copilot.LetStop()
			}
			return copilot.HaltAndContinue(verdict.Feedback)
		})
	})
}

// =============================================================================
// BeforeToolCall — unified pre-execution handler
// =============================================================================

// ToolKind classifies what kind of action is being gated.
type ToolKind string

const (
	ToolKindShell   ToolKind = "shell"   // terminal / bash command
	ToolKindMCP     ToolKind = "mcp"     // MCP protocol tool
	ToolKindBuiltin ToolKind = "builtin" // agent's built-in tool (Read, Write, etc.)
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
//   - VS Code Copilot → "copilot-pre-tool-use"    (Bash + mcp__* tools)
func BeforeToolCall(fn ToolCallFunc) {
	// Claude Code — covers Bash and mcp__* tools via PreToolUse
	AddRoute("claude-pre-tool-use", func() {
		Process(func(ev claude.PreToolUseEvent) claude.PreToolUseResult {
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
	AddRoute("cursor-before-shell", func() {
		Process(func(ev cursor.ShellPreEvent) cursor.ShellPreResult {
			verdict := fn(ToolCallEvent{
				Agent: AgentCursor, Kind: ToolKindShell,
				Command: ev.Command, WorkDir: ev.WorkDir, Raw: &ev,
			})
			return cursorPermissionResult(verdict)
		})
	})

	// Cursor — MCP
	AddRoute("cursor-before-mcp", func() {
		Process(func(ev cursor.MCPPreEvent) cursor.MCPPreResult {
			verdict := fn(ToolCallEvent{
				Agent: AgentCursor, Kind: ToolKindMCP,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput,
				ServerURL: ev.ServerURL, Command: ev.Command, Raw: &ev,
			})
			return cursorPermissionResult(verdict)
		})
	})

	// Windsurf — shell (blocking via exit code 2)
	AddRoute("windsurf-pre-run-command", func() {
		ProcessE(func(ev windsurf.PreRunCommandEvent) (windsurf.PreRunCommandResult, error) {
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
	AddRoute("windsurf-pre-mcp-tool-use", func() {
		ProcessE(func(ev windsurf.PreMCPToolUseEvent) (windsurf.PreMCPToolUseResult, error) {
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

	// Factory Droid — Bash and mcp__* (blocking via exit code 2)
	AddRoute("droid-pre-tool-use", func() {
		ProcessE(func(ev droid.PreToolUseEvent) (droid.PreToolUseResult, error) {
			kind, cmd := droidToolKind(ev.ToolName, ev.ToolInput)
			verdict := fn(ToolCallEvent{
				Agent: AgentDroid, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			if !verdict.Permit {
				return droid.PreToolUseResult{}, errors.New(verdict.Message)
			}
			if verdict.Message != "" {
				return droid.ApproveToolUse(), nil
			}
			return droid.ApproveToolUse(), nil
		})
	})

	// Gemini CLI — all tools (JSON deny response)
	AddRoute("gemini-before-tool", func() {
		Process(func(ev gemini.BeforeToolEvent) gemini.BeforeToolResult {
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

	// VS Code Copilot — Bash + mcp__* via PreToolUse (decision priority: deny > ask > allow)
	AddRoute("copilot-pre-tool-use", func() {
		Process(func(ev copilot.PreToolUseEvent) copilot.PreToolUseResult {
			kind, cmd := copilotToolKind(ev.ToolName, ev.ToolInput)
			verdict := fn(ToolCallEvent{
				Agent: AgentCopilot, Kind: kind, Command: cmd, WorkDir: ev.WorkDir,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			if verdict.Permit {
				if verdict.Message != "" {
					return copilot.ApproveToolUseWithNote(verdict.Message)
				}
				return copilot.ApproveToolUse()
			}
			if verdict.NeedsConfirm {
				return copilot.AskUserAboutTool(verdict.Message)
			}
			return copilot.DenyToolUse(verdict.Message)
		})
	})
}

// =============================================================================
// AfterFileWrite — unified post-file-edit handler
// =============================================================================

// FileDiff represents a single text replacement in a file write operation.
type FileDiff struct {
	Before string // original content (empty for full-file writes)
	After  string // new content
}

// FileWriteEvent provides a unified view of post-file-edit events.
type FileWriteEvent struct {
	Agent     AgentID
	SessionID string
	FilePath  string
	Changes   []FileDiff
	WorkDir   string
	Raw       any
}

// FileWriteVerdict is the decision returned by an AfterFileWrite handler.
type FileWriteVerdict struct {
	Reject   bool   // true = inject feedback into the agent (Claude/Droid only)
	Feedback string // message sent to the agent when Reject is true
	Footnote string // additional context appended after the edit (Claude/Droid only)
}

// AcceptWrite acknowledges the file write with no feedback.
func AcceptWrite() FileWriteVerdict { return FileWriteVerdict{} }

// RejectWrite injects a rejection message into the agent after the write.
func RejectWrite(reason string) FileWriteVerdict { return FileWriteVerdict{Reject: true, Feedback: reason} }

// AnnotateWrite appends a note the agent will see after the write.
func AnnotateWrite(note string) FileWriteVerdict { return FileWriteVerdict{Footnote: note} }

// FileWriteFunc is the handler signature for AfterFileWrite.
type FileWriteFunc func(FileWriteEvent) FileWriteVerdict

// AfterFileWrite registers a unified handler for post-file-edit events on all platforms:
//   - Claude Code   → "claude-after-file-write"  (PostToolUse for Write/Edit tools)
//   - Cursor        → "cursor-after-file-edit"   (fire-and-forget)
//   - Windsurf      → "windsurf-post-write-code" (fire-and-forget)
//   - Factory Droid → "droid-after-file-write"   (PostToolUse for Write/Edit tools)
//   - Gemini CLI    → "gemini-after-file-tool"   (AfterTool for Write/Edit tools)
//   - VS Code Copilot → "copilot-after-file-write" (PostToolUse for Write/Edit tools)
func AfterFileWrite(fn FileWriteFunc) {
	// Claude Code — PostToolUse filtered to Write and Edit tools
	AddRoute("claude-after-file-write", func() {
		Process(func(ev claude.PostToolUseEvent) claude.PostToolUseResult {
			if ev.ToolName != "Write" && ev.ToolName != "Edit" && ev.ToolName != "MultiEdit" {
				return claude.AcknowledgeToolUse()
			}
			changes := claudeWriteChanges(ev.ToolName, ev.ToolInput)
			verdict := fn(FileWriteEvent{
				Agent: AgentClaude, SessionID: ev.SessionID,
				FilePath: claudeFilePath(ev.ToolInput), Changes: changes, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if verdict.Reject {
				return claude.RejectToolResult(verdict.Feedback)
			}
			if verdict.Footnote != "" {
				return claude.AddToolContext(verdict.Footnote)
			}
			return claude.AcknowledgeToolUse()
		})
	})

	// Cursor — afterFileEdit (fire-and-forget)
	AddRoute("cursor-after-file-edit", func() {
		Process(func(ev cursor.FileEditEvent) cursor.FileEditResult {
			changes := make([]FileDiff, len(ev.Edits))
			for i, e := range ev.Edits {
				changes[i] = FileDiff{Before: e.OldText, After: e.NewText}
			}
			fn(FileWriteEvent{
				Agent: AgentCursor, SessionID: ev.ConversationID,
				FilePath: ev.FilePath, Changes: changes, Raw: &ev,
			})
			return cursor.FileEditResult{}
		})
	})

	// Windsurf — post_write_code (fire-and-forget)
	AddRoute("windsurf-post-write-code", func() {
		Process(func(ev windsurf.PostWriteCodeEvent) windsurf.PostWriteCodeResult {
			changes := make([]FileDiff, len(ev.ToolInfo.Edits))
			for i, e := range ev.ToolInfo.Edits {
				changes[i] = FileDiff{Before: e.OldText, After: e.NewText}
			}
			fn(FileWriteEvent{
				Agent: AgentWindsurf, SessionID: ev.TrajectoryID,
				FilePath: ev.ToolInfo.FilePath, Changes: changes, Raw: &ev,
			})
			return windsurf.PostWriteCodeResult{}
		})
	})

	// Factory Droid — PostToolUse filtered to Write and Edit tools
	AddRoute("droid-after-file-write", func() {
		Process(func(ev droid.PostToolUseEvent) droid.PostToolUseResult {
			if ev.ToolName != "Write" && ev.ToolName != "Edit" && ev.ToolName != "MultiEdit" {
				return droid.AcknowledgeToolUse()
			}
			changes := droidWriteChanges(ev.ToolName, ev.ToolInput)
			verdict := fn(FileWriteEvent{
				Agent: AgentDroid, SessionID: ev.SessionID,
				FilePath: droidFilePath(ev.ToolInput), Changes: changes, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if verdict.Reject {
				return droid.RejectToolResult(verdict.Feedback)
			}
			if verdict.Footnote != "" {
				return droid.AddToolContext(verdict.Footnote)
			}
			return droid.AcknowledgeToolUse()
		})
	})

	// Gemini CLI — AfterTool filtered to Write/Edit tools
	AddRoute("gemini-after-file-tool", func() {
		Process(func(ev gemini.AfterToolEvent) gemini.AfterToolResult {
			if ev.ToolName != "write_file" && ev.ToolName != "replace_in_file" &&
				ev.ToolName != "Write" && ev.ToolName != "Edit" {
				return gemini.AcknowledgeToolCall()
			}
			var toolInputFields struct {
				FilePath string `json:"file_path"`
				Path     string `json:"path"`
			}
			json.Unmarshal(ev.ToolInput, &toolInputFields) //nolint:errcheck
			filePath := toolInputFields.FilePath
			if filePath == "" {
				filePath = toolInputFields.Path
			}
			verdict := fn(FileWriteEvent{
				Agent: AgentGemini, SessionID: ev.SessionID,
				FilePath: filePath, WorkDir: ev.WorkDir, Raw: &ev,
			})
			if verdict.Footnote != "" {
				return gemini.AddToolAnnotation(verdict.Footnote)
			}
			return gemini.AcknowledgeToolCall()
		})
	})

	// VS Code Copilot — PostToolUse filtered to Write and Edit tools
	AddRoute("copilot-after-file-write", func() {
		Process(func(ev copilot.PostToolUseEvent) copilot.PostToolUseResult {
			if ev.ToolName != "Write" && ev.ToolName != "Edit" && ev.ToolName != "MultiEdit" {
				return copilot.AcknowledgeToolUse()
			}
			changes := copilotWriteChanges(ev.ToolName, ev.ToolInput)
			verdict := fn(FileWriteEvent{
				Agent: AgentCopilot, SessionID: ev.SessionID,
				FilePath: copilotFilePath(ev.ToolInput), Changes: changes, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if verdict.Reject {
				return copilot.RejectToolResult(verdict.Feedback)
			}
			if verdict.Footnote != "" {
				return copilot.AddToolContext(verdict.Footnote)
			}
			return copilot.AcknowledgeToolUse()
		})
	})
}

// =============================================================================
// BeforePrompt — unified prompt-submit handler
// =============================================================================

// PromptEvent provides a unified view of user-prompt-submission events.
type PromptEvent struct {
	Agent     AgentID
	SessionID string
	Text      string // the prompt text
	Raw       any
}

// PromptVerdict is the decision returned by a BeforePrompt handler.
type PromptVerdict struct {
	Accept  bool
	Message string // shown to user when Accept is false; injected as context when Accept is true
}

// AcceptPrompt allows the prompt through.
func AcceptPrompt() PromptVerdict { return PromptVerdict{Accept: true} }

// RejectPrompt blocks the prompt submission and shows a message to the user.
func RejectPrompt(msg string) PromptVerdict { return PromptVerdict{Accept: false, Message: msg} }

// EnrichPrompt allows the prompt and injects additional context into the agent.
func EnrichPrompt(ctx string) PromptVerdict { return PromptVerdict{Accept: true, Message: ctx} }

// PromptFunc is the handler signature for BeforePrompt.
type PromptFunc func(PromptEvent) PromptVerdict

// BeforePrompt registers a unified handler for prompt-submission events on all platforms:
//   - Claude Code   → "claude-user-prompt-submit"
//   - Cursor        → "cursor-before-submit-prompt"
//   - Windsurf      → "windsurf-pre-user-prompt"   (blocking via exit 2)
//   - Factory Droid → "droid-user-prompt-submit"
//   - Gemini CLI    → "gemini-before-agent"
//   - VS Code Copilot → "copilot-user-prompt-submit"
func BeforePrompt(fn PromptFunc) {
	AddRoute("claude-user-prompt-submit", func() {
		Process(func(ev claude.UserPromptSubmitEvent) claude.UserPromptSubmitResult {
			verdict := fn(PromptEvent{
				Agent: AgentClaude, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
			})
			if !verdict.Accept {
				return claude.RejectPrompt(verdict.Message)
			}
			if verdict.Message != "" {
				return claude.AppendToPrompt(verdict.Message)
			}
			return claude.ApprovePrompt()
		})
	})

	AddRoute("cursor-before-submit-prompt", func() {
		Process(func(ev cursor.PromptPreEvent) cursor.PromptPreResult {
			verdict := fn(PromptEvent{
				Agent: AgentCursor, SessionID: ev.ConversationID, Text: ev.Prompt, Raw: &ev,
			})
			if !verdict.Accept {
				return cursor.BlockPrompt(verdict.Message)
			}
			return cursor.AcceptPrompt()
		})
	})

	AddRoute("windsurf-pre-user-prompt", func() {
		ProcessE(func(ev windsurf.PreUserPromptEvent) (windsurf.PreUserPromptResult, error) {
			verdict := fn(PromptEvent{
				Agent: AgentWindsurf, SessionID: ev.TrajectoryID,
				Text: ev.ToolInfo.UserPrompt, Raw: &ev,
			})
			if !verdict.Accept {
				return windsurf.PreUserPromptResult{}, errors.New(verdict.Message)
			}
			return windsurf.AllowPrompt(), nil
		})
	})

	AddRoute("droid-user-prompt-submit", func() {
		Process(func(ev droid.UserPromptSubmitEvent) droid.UserPromptSubmitResult {
			verdict := fn(PromptEvent{
				Agent: AgentDroid, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
			})
			if !verdict.Accept {
				return droid.RejectPrompt(verdict.Message)
			}
			if verdict.Message != "" {
				return droid.AppendToPrompt(verdict.Message)
			}
			return droid.ApprovePrompt()
		})
	})

	AddRoute("gemini-before-agent", func() {
		Process(func(ev gemini.BeforeAgentEvent) gemini.BeforeAgentResult {
			verdict := fn(PromptEvent{
				Agent: AgentGemini, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
			})
			if !verdict.Accept {
				return gemini.RejectTurn(verdict.Message)
			}
			if verdict.Message != "" {
				return gemini.EnrichTurn(verdict.Message)
			}
			return gemini.AcceptTurn()
		})
	})

	AddRoute("copilot-user-prompt-submit", func() {
		Process(func(ev copilot.UserPromptSubmitEvent) copilot.UserPromptSubmitResult {
			verdict := fn(PromptEvent{
				Agent: AgentCopilot, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
			})
			if !verdict.Accept {
				return copilot.RejectPrompt(verdict.Message)
			}
			if verdict.Message != "" {
				return copilot.AppendToPrompt(verdict.Message)
			}
			return copilot.ApprovePrompt()
		})
	})
}

// =============================================================================
// Internal helpers
// =============================================================================

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

func droidToolKind(toolName string, input json.RawMessage) (ToolKind, string) {
	return claudeToolKind(toolName, input) // same tool naming convention
}

func geminiToolKind(toolName string) ToolKind {
	if len(toolName) >= 5 && toolName[:5] == "mcp__" {
		return ToolKindMCP
	}
	if toolName == "execute_bash" || toolName == "run_shell_command" {
		return ToolKindShell
	}
	return ToolKindBuiltin
}

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

func claudeFilePath(input json.RawMessage) string {
	var v struct {
		FilePath string `json:"file_path"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	return v.FilePath
}

func droidFilePath(input json.RawMessage) string {
	return claudeFilePath(input)
}

func claudeWriteChanges(toolName string, input json.RawMessage) []FileDiff {
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

func droidWriteChanges(toolName string, input json.RawMessage) []FileDiff {
	return claudeWriteChanges(toolName, input)
}

// Copilot uses the same Bash/mcp__ tool naming convention as Claude.
func copilotToolKind(toolName string, input json.RawMessage) (ToolKind, string) {
	return claudeToolKind(toolName, input)
}

func copilotFilePath(input json.RawMessage) string {
	return claudeFilePath(input)
}

func copilotWriteChanges(toolName string, input json.RawMessage) []FileDiff {
	return claudeWriteChanges(toolName, input)
}

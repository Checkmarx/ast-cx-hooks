package cursor

import (
	"encoding/json"
	"strings"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// This file owns the translation between Cursor's wire types and the unified
// hookcore vocabulary. The root agenthooks package wires these adapters into a
// registry; the per-platform logic lives here for locality.

// permissionResult maps a unified ToolVerdict onto Cursor's PermissionResult.
// Cursor does not support input rewriting on these gates, so RewrittenInput is
// ignored here.
func permissionResult(v hookcore.ToolVerdict) PermissionResult {
	if v.Context != "" {
		// Cursor's gate output (permission / user_message / agent_message) has no
		// additionalContext field in the doc, so the unified Context cannot be
		// delivered here per the documentation; report the drop rather than
		// repurpose user_message/agent_message (which carry the deny reason).
		hookcore.LogIgnoredVerdict(hookcore.AgentCursor, "tool-gate", "Context")
	}
	if v.Permit {
		if v.Message != "" {
			return PermitWithNote(v.Message)
		}
		return Permit()
	}
	if v.NeedsConfirm {
		return RequestConfirmation(v.Message, v.Message)
	}
	return Forbid(v.Message, v.Message)
}

// IdleAdapter handles the unified "agent finished" hook for Cursor (stop).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "cursor-stop", func() {
		hookcore.Run(func(ev StopEvent) StopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				CompletionStatus: ev.Status, AutoRetryCount: ev.LoopCount, Raw: &ev,
			})
			if v.Proceed {
				return LetStop()
			}
			return SendFollowup(v.Feedback)
		})
	}
}

// SubagentIdleAdapter handles the unified "subagent finished" hook for Cursor.
func SubagentIdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "cursor-subagent-stop", func() {
		hookcore.Run(func(ev SubagentStopEvent) SubagentStopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				CompletionStatus: ev.Status, AutoRetryCount: ev.LoopCount, Raw: &ev,
			})
			if v.Proceed {
				return LetSubagentStop()
			}
			return SendSubagentFollowup(v.Feedback)
		})
	}
}

// ShellToolAdapter handles the unified pre-tool-call hook for Cursor shell executions.
func ShellToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "cursor-before-shell", func() {
		hookcore.Run(func(ev ShellPreEvent) ShellPreResult {
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCursor, Kind: hookcore.ToolKindShell,
				Command: ev.Command, WorkDir: ev.WorkDir, Raw: &ev,
			})
			return permissionResult(v)
		})
	}
}

// MCPToolAdapter handles the unified pre-tool-call hook for Cursor MCP executions.
func MCPToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "cursor-before-mcp", func() {
		hookcore.Run(func(ev MCPPreEvent) MCPPreResult {
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCursor, Kind: hookcore.ToolKindMCP,
				// ToolInput arrives as a JSON-encoded string; expose it to handlers as
				// the underlying JSON (ToolArgs is raw JSON, not a quoted string).
				ToolName: ev.ToolName, ToolArgs: json.RawMessage(ev.ToolInput),
				ServerURL: ev.ServerURL, Command: ev.Command, Raw: &ev,
			})
			return permissionResult(v)
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Cursor via the
// generic postToolUse hook scoped to the Write tool. Cursor's dedicated afterFileEdit
// hook is fire-and-forget (its result is empty, so no verdict can be delivered),
// whereas postToolUse carries additional_context — so a reject/annotate verdict
// actually reaches the agent. postToolUse cannot BLOCK a completed tool (its output
// is only updated_mcp_tool_output + additional_context), so a unified Reject surfaces
// as additional_context guidance, not a hard block (the write already landed).
// FilePath comes from tool_input.file_path (Cursor's documented path key); the Write
// tool_input content schema is undocumented, so Changes is left empty and the raw
// tool_input is exposed via Raw. Non-Write tools are a no-op.
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "cursor-after-file-edit", func() {
		hookcore.Run(func(ev ToolPostEvent) ToolPostResult {
			if ev.ToolName != "Write" {
				return ToolPostResult{}
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: writeFilePath(ev.ToolInput), WorkDir: ev.WorkDir, Raw: &ev,
			})
			if v.Reject {
				ctx := v.Feedback
				if v.Context != "" {
					ctx = strings.TrimSpace(v.Feedback + "\n" + v.Context)
				}
				return AddContext(ctx)
			}
			if v.Footnote != "" {
				return AddContext(v.Footnote)
			}
			if v.Context != "" {
				return AddContext(v.Context)
			}
			return ToolPostResult{}
		})
	}
}

// PromptAdapter handles the unified prompt-submit hook for Cursor.
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "cursor-before-submit-prompt", func() {
		hookcore.Run(func(ev PromptPreEvent) PromptPreResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				Text: ev.Prompt, Raw: &ev,
			})
			if !v.Accept {
				return BlockPrompt(v.Message)
			}
			return AcceptPrompt()
		})
	}
}

// ToolFailureAdapter handles the unified post-tool-failure hook for Cursor (observational).
func ToolFailureAdapter(fn hookcore.ToolFailureFunc) (string, func()) {
	return "cursor-post-tool-use-failure", func() {
		hookcore.Run(func(ev ToolFailureEvent) ToolFailureResult {
			fn(hookcore.ToolFailureEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				ToolName: ev.ToolName, Error: ev.ErrorMessage, WorkDir: ev.WorkDir, Raw: &ev,
			})
			return ToolFailureResult{}
		})
	}
}

// toolPreDecision maps a unified PreToolUse-style verdict onto Cursor's generic
// preToolUse output (ToolPreResult). Cursor's preToolUse output has no
// additionalContext field, so Context is dropped+logged. Cursor accepts but does
// NOT enforce permission "ask" on preToolUse, so an ask collapses to a deny — a
// gate must fail safe rather than silently allow.
func toolPreDecision(v hookcore.ToolVerdict) ToolPreResult {
	if v.Context != "" {
		hookcore.LogIgnoredVerdict(hookcore.AgentCursor, "pre-tool-use", "Context (preToolUse output has no additionalContext field)")
	}
	if v.Permit {
		if v.RewrittenInput != nil {
			return ToolPreResult{Permission: "allow", UpdatedInput: v.RewrittenInput}
		}
		if v.Message != "" {
			return ToolPreResult{Permission: "allow", AgentNote: v.Message}
		}
		return ToolPreResult{Permission: "allow"}
	}
	return ToolPreResult{Permission: "deny", UserNote: v.Message, AgentNote: v.Message}
}

// writeFilePath extracts the edited file path from a Write tool's input. Cursor
// reports paths as file_path across its hooks (afterFileEdit, beforeReadFile,
// attachments), so the same key is used here.
func writeFilePath(input json.RawMessage) string {
	var v struct {
		FilePath string `json:"file_path"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck // absent path is fine (best-effort)
	return v.FilePath
}

// FileEditAdapter handles the unified pre-file-write GATE for Cursor via the generic
// preToolUse hook scoped to the Write tool. Cursor's preToolUse can DENY a tool
// before it runs (permission:"deny"), so a security scan can block a vulnerable
// write before it lands; non-Write tools are approved untouched so a generic
// pre-tool-call gate (beforeShellExecution / beforeMCPExecution) owns them. The
// Write tool_input content schema is undocumented, so Changes is left empty and the
// raw tool_input is exposed via Raw for handlers that need the proposed content.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "cursor-before-file-write", func() {
		hookcore.Run(func(ev ToolPreEvent) ToolPreResult {
			if ev.ToolName != "Write" {
				return ToolPreResult{Permission: "allow"}
			}
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: writeFilePath(ev.ToolInput), WorkDir: ev.WorkDir, Raw: &ev,
			})
			return toolPreDecision(v)
		})
	}
}

// FileReadAsEditAdapter routes Cursor's beforeReadFile event through the unified
// BeforeFileEdit handler — a single pre-file handler that gates both writes (Changes
// populated) and reads (Changes empty, FilePath set). This matches consumers (e.g.
// ast-cli) that register only BeforeFileEdit and scan files about to be read for
// secrets. allow→permit the read, deny→forbid it; Cursor's beforeReadFile output has
// no additionalContext field, so a unified Context is dropped+logged.
func FileReadAsEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "cursor-before-file-read", func() {
		hookcore.Run(func(ev ReadFilePreEvent) ReadFilePreResult {
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: ev.FilePath, Raw: &ev,
			})
			if v.Context != "" {
				hookcore.LogIgnoredVerdict(hookcore.AgentCursor, "before-read-file", "Context (beforeReadFile output has no additionalContext field)")
			}
			if v.Permit {
				return PermitRead()
			}
			return ForbidRead(v.Message)
		})
	}
}

// FileReadAdapter handles the unified pre-file-read hook for Cursor (beforeReadFile).
func FileReadAdapter(fn hookcore.FileReadFunc) (string, func()) {
	return "cursor-before-read-file", func() {
		hookcore.Run(func(ev ReadFilePreEvent) ReadFilePreResult {
			v := fn(hookcore.FileReadEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: ev.FilePath, Raw: &ev,
			})
			if v.Permit {
				return PermitRead()
			}
			return ForbidRead(v.Message)
		})
	}
}

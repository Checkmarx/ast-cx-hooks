package cursor

import (
	"encoding/json"
	"strings"

	"github.com/Checkmarx/ast-cx-hooks/internal/hookcore"
)

// This file owns the translation between Cursor's wire types and the unified
// hookcore vocabulary. The root agenthooks package wires these adapters into a
// registry; the per-platform logic lives here for locality.

// permissionResult maps a unified ToolVerdict onto Cursor's PermissionResult.
// Cursor has no additionalContext field on permission-style hooks, so remediation
// Context is folded into agent_message.
func permissionResult(v hookcore.ToolVerdict) PermissionResult {
	if v.Permit {
		if msg := agentMessageFromVerdict(v); msg != "" {
			if v.Message != "" && v.Context != "" {
				return PermissionResult{Permission: "allow", UserNote: v.Message, AgentNote: msg}
			}
			return PermissionResult{Permission: "allow", AgentNote: msg}
		}
		return Permit()
	}
	if v.NeedsConfirm {
		return RequestConfirmation(v.Message, agentMessageFromVerdict(v))
	}
	return Forbid(v.Message, formatDenyAgentMessage(agentMessageFromVerdict(v)))
}

// IdleAdapter handles the unified "agent finished" hook for Cursor (stop).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "cursor-stop", func() {
		hookcore.RunFailOpen(func(ev StopEvent) StopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				CompletionStatus: ev.Status, AutoRetryCount: ev.LoopCount, Raw: &ev,
			})
			if v.Proceed {
				return LetStop()
			}
			return SendFollowup(AgentMessageWithContext(v.Feedback, v.Context))
		}, LetStop())
	}
}

// SubagentIdleAdapter handles the unified "subagent finished" hook for Cursor.
func SubagentIdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "cursor-subagent-stop", func() {
		hookcore.RunFailOpen(func(ev SubagentStopEvent) SubagentStopResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				CompletionStatus: ev.Status, AutoRetryCount: ev.LoopCount, Raw: &ev,
			})
			if v.Proceed {
				return LetSubagentStop()
			}
			return SendSubagentFollowup(AgentMessageWithContext(v.Feedback, v.Context))
		}, LetSubagentStop())
	}
}

// ShellToolAdapter handles the unified pre-tool-call hook for Cursor shell executions.
func ShellToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "cursor-before-shell", func() {
		hookcore.RunFailOpen(func(ev ShellPreEvent) ShellPreResult {
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCursor, Kind: hookcore.ToolKindShell,
				Command: ev.Command, WorkDir: ev.WorkDir, Raw: &ev,
			})
			return permissionResult(v)
		}, Permit())
	}
}

// MCPToolAdapter handles the unified pre-tool-call hook for Cursor MCP executions.
func MCPToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "cursor-before-mcp", func() {
		hookcore.RunFailOpen(func(ev MCPPreEvent) MCPPreResult {
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentCursor, Kind: hookcore.ToolKindMCP,
				// ToolInput arrives as a JSON-encoded string; expose it to handlers as
				// the underlying JSON (ToolArgs is raw JSON, not a quoted string).
				ToolName: ev.ToolName, ToolArgs: json.RawMessage(ev.ToolInput),
				ServerURL: ev.ServerURL, Command: ev.Command, Raw: &ev,
			})
			return permissionResult(v)
		}, Permit())
	}
}

// normalizeWorkDir canonicalizes a workDir/workspace-root value Cursor reports on
// Windows. Cursor spells a Windows workspace root as "/c:/foo/bar" (a leading slash
// before the drive letter) rather than the native "c:/foo/bar" or "c:\foo\bar".
// Passed through unchanged, that leading slash survives into every downstream
// consumer of WorkDir — most visibly the `--ignored-file-path`/`--data @<file>`
// arguments ast-cli's ASCA/KICS/SCA guardrails render into suggested
// `cx ignore-vulnerability` commands, and the path ast-cli's ignore-file reader
// actually opens — where Go's os.ReadFile/os.Open reject it outright ("The
// filename, directory name, or volume label syntax is incorrect"). Stripping the
// leading slash here, once, at ingestion, fixes every downstream consumer instead
// of requiring each one to defend against Cursor's spelling individually. Other
// agents' WorkDir/cwd values are already native and pass through unchanged. Used
// by both FileWriteAdapter (below) and FileEditAdapter.
func normalizeWorkDir(path string) string {
	r := strings.ReplaceAll(path, "\\", "/")
	if len(r) >= 3 && r[0] == '/' && isASCIIDriveLetter(r[1]) && r[2] == ':' {
		r = r[1:]
	}
	return r
}

func isASCIIDriveLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// FileWriteAdapter handles the unified post-file-write hook for Cursor via the
// generic postToolUse hook scoped to the Write tool. Cursor's dedicated afterFileEdit
// hook is fire-and-forget (its result is empty, so no verdict can be delivered),
// whereas postToolUse carries additional_context — so a reject/annotate verdict
// actually reaches the agent. postToolUse cannot BLOCK a completed tool (its output
// is only updated_mcp_tool_output + additional_context), so a unified Reject surfaces
// as additional_context guidance, not a hard block (the write already landed).
// FilePath and proposed content come from tool_input via CursorTools.
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "cursor-after-file-edit", func() {
		hookcore.RunFailOpen(func(ev ToolPostEvent) ToolPostResult {
			if !hookcore.CursorTools.IsWrite(ev.ToolName) {
				return ToolPostResult{}
			}
			// Mirror the FileEditAdapter fallback: Cursor may omit cwd on postToolUse
			// events too, so use workspace_roots[0] to anchor --ignored-file-path.
			workDir := ev.WorkDir
			if workDir == "" && len(ev.WorkspaceRoots) > 0 {
				workDir = ev.WorkspaceRoots[0]
			}
			workDir = normalizeWorkDir(workDir)
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: hookcore.CursorTools.FilePath(ev.ToolInput),
				Changes:  hookcore.CursorTools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  workDir, Raw: &ev,
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
		}, ToolPostResult{})
	}
}

// PromptAdapter handles the unified prompt-submit hook for Cursor.
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "cursor-before-submit-prompt", func() {
		hookcore.RunFailOpen(func(ev PromptPreEvent) PromptPreResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				Text: ev.Prompt, Raw: &ev,
			})
			if !v.Accept {
				return BlockPrompt(v.Message)
			}
			return AcceptPrompt()
		}, AcceptPrompt())
	}
}

// ToolFailureAdapter handles the unified post-tool-failure hook for Cursor (observational).
func ToolFailureAdapter(fn hookcore.ToolFailureFunc) (string, func()) {
	return "cursor-post-tool-use-failure", func() {
		hookcore.RunFailOpen(func(ev ToolFailureEvent) ToolFailureResult {
			fn(hookcore.ToolFailureEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				ToolName: ev.ToolName, Error: ev.ErrorMessage, WorkDir: ev.WorkDir, Raw: &ev,
			})
			return ToolFailureResult{}
		}, ToolFailureResult{})
	}
}

// toolPreDecision maps a unified PreToolUse-style verdict onto Cursor's generic
// preToolUse output (ToolPreResult). Cursor has no additionalContext field, so
// remediation Context is folded into agent_message. Cursor accepts but does NOT
// enforce permission "ask" on preToolUse, so an ask collapses to a deny — a gate
// must fail safe rather than silently allow.
func toolPreDecision(v hookcore.ToolVerdict) ToolPreResult {
	if v.Permit {
		if v.RewrittenInput != nil {
			out := ToolPreResult{Permission: "allow", UpdatedInput: v.RewrittenInput}
			if msg := agentMessageFromVerdict(v); msg != "" {
				out.AgentNote = msg
			}
			return out
		}
		if msg := agentMessageFromVerdict(v); msg != "" {
			return ToolPreResult{Permission: "allow", AgentNote: msg}
		}
		return ToolPreResult{Permission: "allow"}
	}
	r := ForbidTool(v.Message, formatDenyAgentMessage(agentMessageFromVerdict(v)))
	r.Context = v.Context
	return r
}

// DenyAgentMessagePrefix is prepended to every Cursor deny agent_message so agents
// reliably recognize hook denies even when the IDE reformats the tool error body.
const DenyAgentMessagePrefix = "CHECKMARX_HOOK_DENY — MANDATORY agent_message (follow exactly; cx-hook-deny rule applies):\n\n"

// formatDenyAgentMessage prefixes remediation guidance for deny responses.
func formatDenyAgentMessage(msg string) string {
	if msg == "" {
		return DenyAgentMessagePrefix + "Follow user_message and invoke the cx-devassist-asca skill for remediation."
	}
	if strings.HasPrefix(msg, DenyAgentMessagePrefix) {
		return msg
	}
	return DenyAgentMessagePrefix + msg
}

// agentMessageFromVerdict builds agent_message for Cursor permission-style hooks.
// When a verdict carries remediation Context, it is appended after the primary message
// so the agent receives the full guidance (Cursor has no additionalContext field).
func agentMessageFromVerdict(v hookcore.ToolVerdict) string {
	if v.Context == "" {
		return v.Message
	}
	if v.Message == "" {
		return v.Context
	}
	return strings.TrimSpace(v.Message + "\n\n" + v.Context)
}

// FileEditAdapter handles the unified pre-file-write GATE for Cursor via the generic
// preToolUse hook scoped to the Write tool. Cursor's preToolUse can DENY a tool
// before it runs (permission:"deny"), so a security scan can block a vulnerable
// write before it lands; non-Write tools are approved untouched so a generic
// pre-tool-call gate (beforeShellExecution / beforeMCPExecution) owns them.
// Proposed content is extracted from tool_input via CursorTools so handlers can
// scan the bytes before they are written.
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "cursor-before-file-write", func() {
		hookcore.RunFailOpen(func(ev ToolPreEvent) ToolPreResult {
			if !hookcore.CursorTools.IsWrite(ev.ToolName) {
				return ToolPreResult{Permission: "allow"}
			}
			// Cursor does not always populate cwd on preToolUse events. Fall back to
			// the first workspace root so --ignored-file-path is always anchored to the
			// repo root in the suppress commands delivered to the agent.
			workDir := ev.WorkDir
			if workDir == "" && len(ev.WorkspaceRoots) > 0 {
				workDir = ev.WorkspaceRoots[0]
			}
			workDir = normalizeWorkDir(workDir)
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: hookcore.CursorTools.FilePath(ev.ToolInput),
				Changes:  hookcore.CursorTools.Changes(ev.ToolName, ev.ToolInput),
				WorkDir:  workDir, Raw: &ev,
			})
			return toolPreDecision(v)
		}, PermitTool())
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
		hookcore.RunFailOpen(func(ev ReadFilePreEvent) ReadFilePreResult {
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: ev.FilePath, Raw: &ev,
			})
			if v.Permit {
				return PermitRead()
			}
			return ForbidRead(AgentMessageWithContext(v.Message, v.Context))
		}, PermitRead())
	}
}

// FileReadAdapter handles the unified pre-file-read hook for Cursor (beforeReadFile).
func FileReadAdapter(fn hookcore.FileReadFunc) (string, func()) {
	return "cursor-before-read-file", func() {
		hookcore.RunFailOpen(func(ev ReadFilePreEvent) ReadFilePreResult {
			v := fn(hookcore.FileReadEvent{
				Agent: hookcore.AgentCursor, SessionID: ev.ConversationID,
				FilePath: ev.FilePath, Raw: &ev,
			})
			if v.Permit {
				return PermitRead()
			}
			return ForbidRead(v.Message)
		}, PermitRead())
	}
}

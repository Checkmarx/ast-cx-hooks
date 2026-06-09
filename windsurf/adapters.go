package windsurf

import (
	"errors"
	"fmt"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// This file owns the translation between Windsurf's Cascade wire types and the
// unified hookcore vocabulary. The root agenthooks package wires these adapters
// into a registry; the per-platform logic lives here for locality.

// IdleAdapter handles the unified "agent finished" hook for Windsurf
// (post_cascade_response). The hook is fire-and-forget: Cascade ignores any
// response, so an Interrupt cannot block.
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "windsurf-post-cascade-response", func() {
		hookcore.Run(func(ev PostCascadeResponseEvent) PostCascadeResponseResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentWindsurf, SessionID: ev.TrajectoryID, Raw: &ev,
			})
			if !v.Proceed {
				hookcore.LogIgnoredVerdict(hookcore.AgentWindsurf, "post-cascade-response", fmt.Sprintf("Interrupt(%q)", v.Feedback))
			}
			return AcknowledgeResponse()
		})
	}
}

// RunCommandAdapter handles the unified pre-tool-call hook for Windsurf shell
// commands (pre_run_command). Blocking via exit code 2.
func RunCommandAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "windsurf-pre-run-command", func() {
		hookcore.RunE(func(ev PreRunCommandEvent) (PreRunCommandResult, error) {
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentWindsurf, Kind: hookcore.ToolKindShell,
				Command: ev.ToolInfo.CommandLine, WorkDir: ev.ToolInfo.WorkDir, Raw: &ev,
			})
			if !v.Permit {
				// Context unsupported on windsurf PreToolUse deny (exit-code-only platform; v.Context ignored).
				return PreRunCommandResult{}, errors.New(v.Message)
			}
			// v.Context (allow) unsupported on windsurf (exit-code-only platform; ignored).
			return AllowCommand(), nil
		})
	}
}

// MCPToolAdapter handles the unified pre-tool-call hook for Windsurf MCP tools
// (pre_mcp_tool_use). Blocking via exit code 2.
func MCPToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "windsurf-pre-mcp-tool-use", func() {
		hookcore.RunE(func(ev PreMCPToolUseEvent) (PreMCPToolUseResult, error) {
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentWindsurf, Kind: hookcore.ToolKindMCP,
				ToolName: ev.ToolInfo.ToolName, ToolArgs: ev.ToolInfo.Arguments,
				ServerURL: ev.ToolInfo.ServerName, Raw: &ev,
			})
			if !v.Permit {
				// Context unsupported on windsurf PreToolUse deny (exit-code-only platform; v.Context ignored).
				return PreMCPToolUseResult{}, errors.New(v.Message)
			}
			// v.Context (allow) unsupported on windsurf (exit-code-only platform; ignored).
			return AllowMCPTool(), nil
		})
	}
}

// FileEditAdapter handles the unified pre-file-write GATE for Windsurf
// (pre_write_code). Fires BEFORE Cascade writes/edits a file and BLOCKS via exit
// code 2 when the verdict denies. Context/ask are unsupported on this exit-code-only
// gate, so a deny carries only its reason (surfaced on stderr).
func FileEditAdapter(fn hookcore.FileEditFunc) (string, func()) {
	return "windsurf-pre-write-code", func() {
		hookcore.RunE(func(ev PreWriteCodeEvent) (PreWriteCodeResult, error) {
			changes := make([]hookcore.FileDiff, len(ev.ToolInfo.Edits))
			for i, e := range ev.ToolInfo.Edits {
				changes[i] = hookcore.FileDiff{Before: e.OldText, After: e.NewText}
			}
			v := fn(hookcore.FileEditEvent{
				Agent: hookcore.AgentWindsurf, SessionID: ev.TrajectoryID,
				FilePath: ev.ToolInfo.FilePath, Changes: changes, Raw: &ev,
			})
			if !v.Permit {
				return PreWriteCodeResult{}, errors.New(v.Message)
			}
			return AllowWrite(), nil
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Windsurf
// (post_write_code). Informational only.
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "windsurf-post-write-code", func() {
		hookcore.Run(func(ev PostWriteCodeEvent) PostWriteCodeResult {
			changes := make([]hookcore.FileDiff, len(ev.ToolInfo.Edits))
			for i, e := range ev.ToolInfo.Edits {
				changes[i] = hookcore.FileDiff{Before: e.OldText, After: e.NewText}
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentWindsurf, SessionID: ev.TrajectoryID,
				FilePath: ev.ToolInfo.FilePath, Changes: changes, Raw: &ev,
			})
			// post_write_code is fire-and-forget on windsurf (exit-code-only platform):
			// an actionable verdict cannot be delivered, so report the drop and move on.
			if v.Reject || v.Footnote != "" || v.Context != "" {
				hookcore.LogIgnoredVerdict(hookcore.AgentWindsurf, "post-write-code", "Reject/Footnote/Context")
			}
			return PostWriteCodeResult{}
		})
	}
}

// PromptAdapter handles the unified prompt-submit hook for Windsurf
// (pre_user_prompt). Blocking via exit code 2.
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "windsurf-pre-user-prompt", func() {
		hookcore.RunE(func(ev PreUserPromptEvent) (PreUserPromptResult, error) {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentWindsurf, SessionID: ev.TrajectoryID,
				Text: ev.ToolInfo.UserPrompt, Raw: &ev,
			})
			if !v.Accept {
				return PreUserPromptResult{}, errors.New(v.Message)
			}
			return AllowPrompt(), nil
		})
	}
}

// FileReadAdapter handles the unified pre-file-read hook for Windsurf
// (pre_read_code). Blocking via exit code 2.
func FileReadAdapter(fn hookcore.FileReadFunc) (string, func()) {
	return "windsurf-pre-read-code", func() {
		hookcore.RunE(func(ev PreReadCodeEvent) (PreReadCodeResult, error) {
			v := fn(hookcore.FileReadEvent{
				Agent: hookcore.AgentWindsurf, SessionID: ev.TrajectoryID,
				FilePath: ev.ToolInfo.FilePath, Raw: &ev,
			})
			if !v.Permit {
				return PreReadCodeResult{}, errors.New(v.Message)
			}
			return AllowRead(), nil
		})
	}
}

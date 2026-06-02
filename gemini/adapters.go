package gemini

import (
	"encoding/json"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// This file owns the translation between Gemini's wire types and the unified
// hookcore vocabulary. The root agenthooks package wires these adapters into a
// registry; the per-platform logic lives here for locality.

// IdleAdapter handles the unified "agent finished" hook for Gemini (AfterAgent).
func IdleAdapter(fn hookcore.AgentIdleFunc) (string, func()) {
	return "gemini-after-agent", func() {
		hookcore.Run(func(ev AfterAgentEvent) AfterAgentResult {
			v := fn(hookcore.AgentIdleEvent{
				Agent: hookcore.AgentGemini, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if !v.Proceed {
				return RetryWithFeedback(v.Feedback)
			}
			return AcceptResponse()
		})
	}
}

// ToolAdapter handles the unified pre-tool-call hook for Gemini (BeforeTool).
func ToolAdapter(fn hookcore.ToolCallFunc) (string, func()) {
	return "gemini-before-tool", func() {
		hookcore.Run(func(ev BeforeToolEvent) BeforeToolResult {
			kind := hookcore.GeminiToolKind(ev.ToolName)
			var cmd string
			if kind == hookcore.ToolKindShell {
				// Gemini's shell tools (execute_bash / run_shell_command) carry the
				// command under "command"; surface it so command-based gating works.
				var c struct {
					Command string `json:"command"`
				}
				json.Unmarshal(ev.ToolInput, &c) //nolint:errcheck // absent command is fine
				cmd = c.Command
			}
			v := fn(hookcore.ToolCallEvent{
				Agent: hookcore.AgentGemini, Kind: kind, Command: cmd,
				ToolName: ev.ToolName, ToolArgs: ev.ToolInput, Raw: &ev,
			})
			if !v.Permit {
				return DenyToolCall(v.Message)
			}
			if v.RewrittenInput != nil {
				return ApproveToolCallWithInput(v.RewrittenInput)
			}
			return ApproveToolCall()
		})
	}
}

// FileWriteAdapter handles the unified post-file-write hook for Gemini (AfterTool write_file/replace_in_file).
func FileWriteAdapter(fn hookcore.FileWriteFunc) (string, func()) {
	return "gemini-after-file-tool", func() {
		hookcore.Run(func(ev AfterToolEvent) AfterToolResult {
			if !hookcore.IsGeminiWriteTool(ev.ToolName) {
				return AcknowledgeToolCall()
			}
			var f struct {
				FilePath string `json:"file_path"`
				Path     string `json:"path"`
			}
			json.Unmarshal(ev.ToolInput, &f) //nolint:errcheck
			filePath := f.FilePath
			if filePath == "" {
				filePath = f.Path
			}
			v := fn(hookcore.FileWriteEvent{
				Agent: hookcore.AgentGemini, SessionID: ev.SessionID,
				FilePath: filePath, WorkDir: ev.WorkDir, Raw: &ev,
			})
			if v.Footnote != "" {
				return AddToolAnnotation(v.Footnote)
			}
			return AcknowledgeToolCall()
		})
	}
}

// PromptAdapter handles the unified prompt-submit hook for Gemini (BeforeAgent).
func PromptAdapter(fn hookcore.PromptFunc) (string, func()) {
	return "gemini-before-agent", func() {
		hookcore.Run(func(ev BeforeAgentEvent) BeforeAgentResult {
			v := fn(hookcore.PromptEvent{
				Agent: hookcore.AgentGemini, SessionID: ev.SessionID, Text: ev.Prompt, Raw: &ev,
			})
			if !v.Accept {
				return RejectTurn(v.Message)
			}
			if v.Message != "" {
				return EnrichTurn(v.Message)
			}
			return AcceptTurn()
		})
	}
}

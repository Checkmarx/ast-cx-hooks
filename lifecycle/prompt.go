package lifecycle

import (
	"errors"

	"github.com/CheckmarxDev/ast-cx-hooks/claude"
	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
	"github.com/CheckmarxDev/ast-cx-hooks/droid"
	"github.com/CheckmarxDev/ast-cx-hooks/gemini"
	"github.com/CheckmarxDev/ast-cx-hooks/internal/dispatch"
	"github.com/CheckmarxDev/ast-cx-hooks/windsurf"
)

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
func BeforePrompt(fn PromptFunc) {
	dispatch.AddRoute("claude-user-prompt-submit", func() {
		dispatch.Process(func(ev claude.UserPromptSubmitEvent) claude.UserPromptSubmitResult {
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

	dispatch.AddRoute("cursor-before-submit-prompt", func() {
		dispatch.Process(func(ev cursor.PromptPreEvent) cursor.PromptPreResult {
			verdict := fn(PromptEvent{
				Agent: AgentCursor, SessionID: ev.ConversationID, Text: ev.Prompt, Raw: &ev,
			})
			if !verdict.Accept {
				return cursor.BlockPrompt(verdict.Message)
			}
			return cursor.AcceptPrompt()
		})
	})

	dispatch.AddRoute("windsurf-pre-user-prompt", func() {
		dispatch.ProcessE(func(ev windsurf.PreUserPromptEvent) (windsurf.PreUserPromptResult, error) {
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

	dispatch.AddRoute("droid-user-prompt-submit", func() {
		dispatch.Process(func(ev droid.UserPromptSubmitEvent) droid.UserPromptSubmitResult {
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

	dispatch.AddRoute("gemini-before-agent", func() {
		dispatch.Process(func(ev gemini.BeforeAgentEvent) gemini.BeforeAgentResult {
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
}

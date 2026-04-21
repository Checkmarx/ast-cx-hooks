package lifecycle

import (
	"errors"
	"fmt"
	"os"

	"github.com/CheckmarxDev/ast-cx-hooks/claude"
	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
	"github.com/CheckmarxDev/ast-cx-hooks/droid"
	"github.com/CheckmarxDev/ast-cx-hooks/gemini"
	"github.com/CheckmarxDev/ast-cx-hooks/internal/dispatch"
	"github.com/CheckmarxDev/ast-cx-hooks/windsurf"
)

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
	case AgentClaude, AgentDroid, AgentGemini:
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
func WhenAgentIdle(fn AgentIdleFunc) {
	dispatch.AddRoute("claude-stop", func() {
		dispatch.Process(func(ev claude.StopEvent) claude.StopResult {
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

	dispatch.AddRoute("cursor-stop", func() {
		dispatch.Process(func(ev cursor.StopEvent) cursor.StopResult {
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

	dispatch.AddRoute("windsurf-post-cascade-response", func() {
		dispatch.Process(func(ev windsurf.PostCascadeResponseEvent) windsurf.PostCascadeResponseResult {
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

	dispatch.AddRoute("droid-stop", func() {
		dispatch.Process(func(ev droid.StopEvent) droid.StopResult {
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

	dispatch.AddRoute("gemini-after-agent", func() {
		// Gemini AfterAgent retry requires exit code 2 (not continue:false which stops the loop).
		// ProcessE writes feedback to stderr and exits 2, causing Gemini to retry with that feedback.
		dispatch.ProcessE(func(ev gemini.AfterAgentEvent) (gemini.AfterAgentResult, error) {
			verdict := fn(AgentIdleEvent{
				Agent: AgentGemini, SessionID: ev.SessionID, WorkDir: ev.WorkDir,
				IsRepeat: ev.HookActive, Raw: &ev,
			})
			if !verdict.Proceed {
				return gemini.AfterAgentResult{}, errors.New(verdict.Feedback)
			}
			return gemini.AcceptResponse(), nil
		})
	})
}

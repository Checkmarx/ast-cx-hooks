package cx

import (
	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
	"github.com/CheckmarxDev/ast-cx-hooks/guardrails"
)

// cxWhenAgentIdle: agent finished its turn. Nothing to enforce yet.
func cxWhenAgentIdle(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
	return agenthooks.Resume()
}

// cxBeforeToolCall gates shell execution against the organization's blocklist.
func cxBeforeToolCall(ev agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
	if !ev.IsShell() {
		return agenthooks.Allow()
	}
	if blocked, reason := guardrails.CheckShellCommand(ev.Command); blocked {
		return agenthooks.Deny(reason)
	}
	return agenthooks.Allow()
}

// cxAfterFileWrite: react to file edits. Nothing to enforce yet.
func cxAfterFileWrite(_ agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
	return agenthooks.AcceptWrite()
}

// cxBeforePrompt runs all prompt guardrails before the prompt reaches the AI agent.
func cxBeforePrompt(ev agenthooks.PromptEvent) agenthooks.PromptVerdict {
	if reason := guardrails.ScanPrompt(ev.Text); reason != "" {
		return agenthooks.RejectPrompt(reason)
	}
	return agenthooks.AcceptPrompt()
}

// RegisterGuardrails wires the four guardrail handlers.
func RegisterGuardrails() {
	agenthooks.WhenAgentIdle(cxWhenAgentIdle)
	agenthooks.BeforeToolCall(cxBeforeToolCall)
	agenthooks.AfterFileWrite(cxAfterFileWrite)
	agenthooks.BeforePrompt(cxBeforePrompt)
}

// RegisterPassThrough wires no-op handlers that always allow the action.
// Used when the license check fails so we still emit valid JSON (fail-open).
func RegisterPassThrough() {
	agenthooks.WhenAgentIdle(func(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.BeforeToolCall(func(_ agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Allow() })
	agenthooks.AfterFileWrite(func(_ agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict { return agenthooks.AcceptWrite() })
	agenthooks.BeforePrompt(func(_ agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.AcceptPrompt() })
}

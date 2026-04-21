// Package agenthooks provides a framework for building hooks for AI coding agents.
//
// It supports Claude Code, Cursor, Windsurf Cascade, Factory Droid, and Gemini CLI
// through a single unified API plus platform-specific packages for advanced use.
//
// Quick start with unified handlers (one handler works across all agents):
//
//	package main
//
//	import "github.com/CheckmarxDev/ast-cx-hooks"
//
//	func main() {
//	    agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
//	        if e.IsLooping() {
//	            return agenthooks.Resume()
//	        }
//	        return agenthooks.Interrupt("Please review changes before finishing.")
//	    })
//	    agenthooks.Dispatch()
//	}
//
// Platform-specific handlers for advanced use:
//
//	agenthooks.AddRoute("claude-pre-tool-use", func() {
//	    agenthooks.Process(func(e claude.PreToolUseEvent) claude.PreToolUseResult {
//	        return claude.ApproveToolUse()
//	    })
//	})
package agenthooks

import (
	"github.com/CheckmarxDev/ast-cx-hooks/internal/dispatch"
	"github.com/CheckmarxDev/ast-cx-hooks/lifecycle"
)

// =============================================================================
// Routing core (re-exported from internal/dispatch)
// =============================================================================

// RouteFunc is the type for handlers registered via AddRoute.
type RouteFunc = dispatch.RouteFunc

// AddRoute registers fn under the given command name.
func AddRoute(name string, fn RouteFunc) { dispatch.AddRoute(name, fn) }

// Dispatch selects and runs the handler whose name matches os.Args[1].
func Dispatch() { dispatch.Dispatch() }

// Process reads JSON from stdin, passes it to handler, and writes the result to stdout.
// Any stdin parse error causes a graceful exit (code 0) so a bad payload never blocks an agent.
func Process[I, O any](handler func(I) O) { dispatch.Process(handler) }

// ProcessE is like Process but allows the handler to signal a blocking error (exit 2).
func ProcessE[I, O any](handler func(I) (O, error)) { dispatch.ProcessE(handler) }

// ClearRoutes removes all registered handlers. Intended for use in tests.
func ClearRoutes() { dispatch.ClearRoutes() }

// =============================================================================
// Agent and tool constants (re-exported from lifecycle)
// =============================================================================

// AgentID identifies which AI coding agent triggered the hook.
type AgentID = lifecycle.AgentID

const (
	AgentClaude   = lifecycle.AgentClaude
	AgentCursor   = lifecycle.AgentCursor
	AgentWindsurf = lifecycle.AgentWindsurf
	AgentDroid    = lifecycle.AgentDroid
	AgentGemini   = lifecycle.AgentGemini
)

// ToolKind classifies what kind of action a BeforeToolCall handler is gating.
type ToolKind = lifecycle.ToolKind

const (
	ToolKindShell   = lifecycle.ToolKindShell
	ToolKindMCP     = lifecycle.ToolKindMCP
	ToolKindBuiltin = lifecycle.ToolKindBuiltin
)

// =============================================================================
// WhenAgentIdle (re-exported from lifecycle)
// =============================================================================

type AgentIdleEvent = lifecycle.AgentIdleEvent
type IdleVerdict = lifecycle.IdleVerdict
type AgentIdleFunc = lifecycle.AgentIdleFunc

// Resume allows the agent to stop normally.
func Resume() IdleVerdict { return lifecycle.Resume() }

// Interrupt prevents the agent from stopping and sends feedback for the next iteration.
func Interrupt(feedback string) IdleVerdict { return lifecycle.Interrupt(feedback) }

// WhenAgentIdle registers a unified handler for "agent finished responding" events
// on all five platforms.
func WhenAgentIdle(fn AgentIdleFunc) { lifecycle.WhenAgentIdle(fn) }

// =============================================================================
// BeforeToolCall (re-exported from lifecycle)
// =============================================================================

type ToolCallEvent = lifecycle.ToolCallEvent
type ToolVerdict = lifecycle.ToolVerdict
type ToolCallFunc = lifecycle.ToolCallFunc

// Allow permits the tool call with no message.
func Allow() ToolVerdict { return lifecycle.Allow() }

// AllowWithNote permits the tool call and surfaces a note.
func AllowWithNote(msg string) ToolVerdict { return lifecycle.AllowWithNote(msg) }

// Deny blocks the tool call and sends a reason to the agent.
func Deny(reason string) ToolVerdict { return lifecycle.Deny(reason) }

// AskUser blocks pending user confirmation and explains why.
func AskUser(reason string) ToolVerdict { return lifecycle.AskUser(reason) }

// BeforeToolCall registers a unified handler for pre-execution events on all platforms.
func BeforeToolCall(fn ToolCallFunc) { lifecycle.BeforeToolCall(fn) }

// =============================================================================
// AfterFileWrite (re-exported from lifecycle)
// =============================================================================

type FileDiff = lifecycle.FileDiff
type FileWriteEvent = lifecycle.FileWriteEvent
type FileWriteVerdict = lifecycle.FileWriteVerdict
type FileWriteFunc = lifecycle.FileWriteFunc

// AcceptWrite acknowledges the file write with no feedback.
func AcceptWrite() FileWriteVerdict { return lifecycle.AcceptWrite() }

// RejectWrite injects a rejection message into the agent after the write.
func RejectWrite(reason string) FileWriteVerdict { return lifecycle.RejectWrite(reason) }

// AnnotateWrite appends a note the agent will see after the write.
func AnnotateWrite(note string) FileWriteVerdict { return lifecycle.AnnotateWrite(note) }

// AfterFileWrite registers a unified handler for post-file-edit events on all platforms.
func AfterFileWrite(fn FileWriteFunc) { lifecycle.AfterFileWrite(fn) }

// =============================================================================
// BeforePrompt (re-exported from lifecycle)
// =============================================================================

type PromptEvent = lifecycle.PromptEvent
type PromptVerdict = lifecycle.PromptVerdict
type PromptFunc = lifecycle.PromptFunc

// AcceptPrompt allows the prompt through.
func AcceptPrompt() PromptVerdict { return lifecycle.AcceptPrompt() }

// RejectPrompt blocks the prompt submission and shows a message to the user.
func RejectPrompt(msg string) PromptVerdict { return lifecycle.RejectPrompt(msg) }

// EnrichPrompt allows the prompt and injects additional context into the agent.
func EnrichPrompt(ctx string) PromptVerdict { return lifecycle.EnrichPrompt(ctx) }

// BeforePrompt registers a unified handler for prompt-submission events on all platforms.
func BeforePrompt(fn PromptFunc) { lifecycle.BeforePrompt(fn) }

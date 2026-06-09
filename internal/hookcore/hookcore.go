// Package hookcore defines the platform-agnostic vocabulary of the unified hook
// API: the event views, verdicts, and constructors shared across every agent.
//
// It is a leaf package (no dependency on the platform packages or the root
// agenthooks package), which lets platform packages and the root share these
// types without an import cycle. The root re-exports everything here via aliases,
// so callers continue to use agenthooks.AgentIdleEvent, agenthooks.Allow, etc.
package hookcore

import "encoding/json"

// AgentID identifies which AI coding agent triggered the hook.
type AgentID string

const (
	AgentClaude     AgentID = "claude"
	AgentCursor     AgentID = "cursor"
	AgentWindsurf   AgentID = "windsurf"
	AgentDroid      AgentID = "droid"
	AgentGemini     AgentID = "gemini"
	AgentCopilot    AgentID = "copilot"     // VS Code GitHub Copilot extension
	AgentCopilotCLI AgentID = "copilot-cli" // GitHub Copilot CLI + cloud agent
)

// --- Idle ---

// AgentIdleEvent provides a unified view of "agent finished responding" events.
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

// AgentIdleFunc is the handler signature for WhenAgentIdle / WhenSubagentIdle.
type AgentIdleFunc func(AgentIdleEvent) IdleVerdict

// --- Tool call ---

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

	// RewrittenInput optionally replaces the tool's input before execution.
	// Honored on Claude, Droid, Gemini, and Copilot; ignored on Cursor/Windsurf.
	RewrittenInput json.RawMessage

	// Context is additional context delivered to the agent via the platform's
	// documented additionalContext field. To stay strictly conformant to each
	// agent's hook documentation, it is emitted ONLY where the doc defines that
	// field for the hook — Claude (PreToolUse/PostToolUse and other context-
	// carrying events), Gemini AfterTool, and Copilot CLI postToolUse/
	// postToolUseFailure. Where the doc defines no additionalContext field for the
	// hook (Cursor gates, Copilot CLI preToolUse, Gemini BeforeTool, Windsurf,
	// Cursor afterFileEdit), it is NOT repurposed into another field — it is dropped
	// and logged via LogIgnoredVerdict so the loss is observable.
	Context string
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

// AllowWithInput permits the tool call but rewrites its input before execution.
// updated is the full replacement tool input as raw JSON. Honored on Claude,
// Droid, Gemini, and Copilot; on Cursor/Windsurf it falls back to a plain allow.
func AllowWithInput(updated json.RawMessage) ToolVerdict {
	return ToolVerdict{Permit: true, RewrittenInput: updated}
}

// AllowWithContext permits the tool call and injects additionalContext for the
// agent. Honored on Claude and Copilot; elsewhere it falls back to a plain allow.
func AllowWithContext(context string) ToolVerdict {
	return ToolVerdict{Permit: true, Context: context}
}

// DenyWithContext blocks the tool call, sends reason to the agent, AND injects
// additionalContext alongside the denial — e.g. an instruction to run a
// remediation skill. The context is honored on platforms whose deny payload
// carries an additionalContext field (Claude, Copilot); elsewhere this behaves
// as a plain Deny(reason).
func DenyWithContext(reason, context string) ToolVerdict {
	return ToolVerdict{Permit: false, Message: reason, Context: context}
}

// ToolPath is the canonical decision a PreToolUse adapter takes for a ToolVerdict.
// Classifying the verdict in one place fixes the field precedence — on allow:
// RewrittenInput > Context > Message; on deny: NeedsConfirm > Context — so the
// Claude-schema adapters (Claude, Copilot, Droid) cannot drift, and makes that
// precedence a single unit-tested seam. A platform that cannot deliver a path
// (e.g. Droid has no additionalContext on PreToolUse) collapses it to the nearest
// supported builder in its own switch.
type ToolPath int

const (
	PathAllow            ToolPath = iota // plain allow
	PathAllowWithInput                   // allow, replacing the tool input
	PathAllowWithContext                 // allow, injecting additionalContext
	PathAllowWithNote                    // allow, with a user-facing note
	PathAsk                              // ask the user to confirm
	PathDeny                             // plain deny
	PathDenyWithContext                  // deny, injecting additionalContext
)

// Path classifies the verdict into its canonical ToolPath. Adapters switch on the
// result instead of re-implementing the precedence tree.
func (v ToolVerdict) Path() ToolPath {
	if v.Permit {
		switch {
		case v.RewrittenInput != nil:
			return PathAllowWithInput
		case v.Context != "":
			return PathAllowWithContext
		case v.Message != "":
			return PathAllowWithNote
		default:
			return PathAllow
		}
	}
	if v.NeedsConfirm {
		return PathAsk
	}
	if v.Context != "" {
		return PathDenyWithContext
	}
	return PathDeny
}

// ToolCallFunc is the handler signature for BeforeToolCall.
type ToolCallFunc func(ToolCallEvent) ToolVerdict

// --- File write ---

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

	// Context is additionalContext injected for the agent alongside a reject —
	// e.g. an instruction to run a remediation skill on the findings that caused
	// the reject. Honored on platforms whose post-write payload carries an
	// additionalContext field (Claude, Copilot, Droid, Gemini); ignored on the
	// fire-and-forget platforms (Cursor, Windsurf).
	Context string
}

// AcceptWrite acknowledges the file write with no feedback.
func AcceptWrite() FileWriteVerdict { return FileWriteVerdict{} }

// RejectWrite injects a rejection message into the agent after the write.
func RejectWrite(reason string) FileWriteVerdict {
	return FileWriteVerdict{Reject: true, Feedback: reason}
}

// RejectWriteWithContext injects a rejection message AND additionalContext after
// the write — e.g. the reason names the findings and the context instructs the
// agent to run a remediation skill. Honored on Claude, Copilot, Droid, and
// Gemini; on the fire-and-forget platforms (Cursor, Windsurf) it behaves as a
// plain RejectWrite(reason).
func RejectWriteWithContext(reason, context string) FileWriteVerdict {
	return FileWriteVerdict{Reject: true, Feedback: reason, Context: context}
}

// AnnotateWrite appends a note the agent will see after the write.
func AnnotateWrite(note string) FileWriteVerdict { return FileWriteVerdict{Footnote: note} }

// FileWriteFunc is the handler signature for AfterFileWrite.
type FileWriteFunc func(FileWriteEvent) FileWriteVerdict

// --- File edit (pre-write gate) ---

// FileEditEvent is the unified view of a pre-file-write event: the agent is about
// to create or modify a file. It shares FileWriteEvent's shape (path plus the
// proposed Before/After changes) so a handler can inspect the content BEFORE it is
// written — e.g. a security scan of the proposed content. Unlike AfterFileWrite,
// which fires post-write and can only annotate, a BeforeFileEdit handler can BLOCK
// the write before it lands.
type FileEditEvent = FileWriteEvent

// FileEditVerdict is the decision returned by a BeforeFileEdit handler. A pre-file
// gate is a PreToolUse-style permission decision (allow / deny / ask, optionally
// carrying additionalContext or a rewritten tool input), so it reuses ToolVerdict
// and its Path() classifier — the same field precedence the tool-call gate uses.
type FileEditVerdict = ToolVerdict

// AcceptEdit allows the file write to proceed.
func AcceptEdit() FileEditVerdict { return Allow() }

// AcceptEditWithInput allows the write but rewrites the tool input first — e.g. an
// auto-remediated version of the content. Honored where updatedInput is (Claude,
// Droid); a plain allow elsewhere.
func AcceptEditWithInput(updated json.RawMessage) FileEditVerdict { return AllowWithInput(updated) }

// RejectEdit blocks the file write before it lands and tells the agent why.
func RejectEdit(reason string) FileEditVerdict { return Deny(reason) }

// RejectEditWithContext blocks the write AND injects additionalContext alongside
// the denial — e.g. the reason names the findings and the context instructs the
// agent to run a remediation skill. The context rides along on platforms whose
// deny payload carries an additionalContext field (Claude); elsewhere it behaves
// as a plain RejectEdit(reason).
func RejectEditWithContext(reason, context string) FileEditVerdict {
	return DenyWithContext(reason, context)
}

// AskBeforeEdit blocks the write pending user confirmation and explains why.
func AskBeforeEdit(reason string) FileEditVerdict { return AskUser(reason) }

// FileEditFunc is the handler signature for BeforeFileEdit.
type FileEditFunc func(FileEditEvent) FileEditVerdict

// --- Prompt ---

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

// --- Tool failure ---

// ToolFailureEvent provides a unified view of "a tool call failed" events.
type ToolFailureEvent struct {
	Agent     AgentID
	SessionID string
	ToolName  string
	Error     string // failure message
	WorkDir   string
	Raw       any
}

// ToolFailureVerdict is the decision returned by an AfterToolFailure handler.
type ToolFailureVerdict struct {
	Reject   bool   // inject feedback into the agent (Claude only)
	Feedback string // message sent to the agent when Reject is true
	Footnote string // additional context appended after the failure (Claude only)
}

// AcknowledgeFailure accepts the failure with no feedback.
func AcknowledgeFailure() ToolFailureVerdict { return ToolFailureVerdict{} }

// AnnotateFailure appends a note the agent sees after the failure (Claude only).
func AnnotateFailure(note string) ToolFailureVerdict { return ToolFailureVerdict{Footnote: note} }

// RejectAfterFailure injects feedback into the agent after the failure (Claude only).
func RejectAfterFailure(reason string) ToolFailureVerdict {
	return ToolFailureVerdict{Reject: true, Feedback: reason}
}

// ToolFailureFunc is the handler signature for AfterToolFailure.
type ToolFailureFunc func(ToolFailureEvent) ToolFailureVerdict

// --- File read ---

// FileReadEvent provides a unified view of "agent about to read a file" events.
type FileReadEvent struct {
	Agent     AgentID
	SessionID string
	FilePath  string
	WorkDir   string
	Raw       any
}

// FileReadVerdict is the decision returned by a BeforeFileRead handler.
type FileReadVerdict struct {
	Permit  bool
	Message string // shown to the user when denied
}

// AllowRead permits the file read.
func AllowRead() FileReadVerdict { return FileReadVerdict{Permit: true} }

// DenyRead blocks the agent from reading the file and explains why.
func DenyRead(reason string) FileReadVerdict { return FileReadVerdict{Permit: false, Message: reason} }

// FileReadFunc is the handler signature for BeforeFileRead.
type FileReadFunc func(FileReadEvent) FileReadVerdict

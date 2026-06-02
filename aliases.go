package agenthooks

import "github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"

// The unified hook vocabulary lives in internal/hookcore (a leaf package the
// platform packages can also import without a cycle) and is re-exported here so
// the public API stays agenthooks.AgentIdleEvent, agenthooks.Allow, and so on.

type (
	AgentID = hookcore.AgentID

	AgentIdleEvent = hookcore.AgentIdleEvent
	IdleVerdict    = hookcore.IdleVerdict
	AgentIdleFunc  = hookcore.AgentIdleFunc

	ToolKind      = hookcore.ToolKind
	ToolCallEvent = hookcore.ToolCallEvent
	ToolVerdict   = hookcore.ToolVerdict
	ToolCallFunc  = hookcore.ToolCallFunc

	FileDiff         = hookcore.FileDiff
	FileWriteEvent   = hookcore.FileWriteEvent
	FileWriteVerdict = hookcore.FileWriteVerdict
	FileWriteFunc    = hookcore.FileWriteFunc

	PromptEvent   = hookcore.PromptEvent
	PromptVerdict = hookcore.PromptVerdict
	PromptFunc    = hookcore.PromptFunc

	ToolFailureEvent   = hookcore.ToolFailureEvent
	ToolFailureVerdict = hookcore.ToolFailureVerdict
	ToolFailureFunc    = hookcore.ToolFailureFunc

	FileReadEvent   = hookcore.FileReadEvent
	FileReadVerdict = hookcore.FileReadVerdict
	FileReadFunc    = hookcore.FileReadFunc
)

const (
	AgentClaude   = hookcore.AgentClaude
	AgentCursor   = hookcore.AgentCursor
	AgentWindsurf = hookcore.AgentWindsurf
	AgentDroid    = hookcore.AgentDroid
	AgentGemini   = hookcore.AgentGemini
	AgentCopilot  = hookcore.AgentCopilot

	ToolKindShell   = hookcore.ToolKindShell
	ToolKindMCP     = hookcore.ToolKindMCP
	ToolKindBuiltin = hookcore.ToolKindBuiltin
)

// Verdict constructors, re-exported from hookcore.
var (
	Resume    = hookcore.Resume
	Interrupt = hookcore.Interrupt

	Allow          = hookcore.Allow
	AllowWithNote  = hookcore.AllowWithNote
	Deny           = hookcore.Deny
	AskUser        = hookcore.AskUser
	AllowWithInput = hookcore.AllowWithInput

	AcceptWrite   = hookcore.AcceptWrite
	RejectWrite   = hookcore.RejectWrite
	AnnotateWrite = hookcore.AnnotateWrite

	AcceptPrompt = hookcore.AcceptPrompt
	RejectPrompt = hookcore.RejectPrompt
	EnrichPrompt = hookcore.EnrichPrompt

	AcknowledgeFailure = hookcore.AcknowledgeFailure
	AnnotateFailure    = hookcore.AnnotateFailure
	RejectAfterFailure = hookcore.RejectAfterFailure

	AllowRead = hookcore.AllowRead
	DenyRead  = hookcore.DenyRead
)

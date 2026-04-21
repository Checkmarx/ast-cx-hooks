// Package lifecycle defines the unified agent lifecycle hook types, events,
// verdicts, and handlers that work across all five AI coding agents
// (Claude Code, Cursor, Windsurf Cascade, Factory Droid, Gemini CLI).
//
// Users typically access this package indirectly via the root agenthooks package:
//
//	import agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
//
// Advanced users may import this package directly to access event types.
package lifecycle

// AgentID identifies which AI coding agent triggered the hook.
type AgentID string

const (
	AgentClaude   AgentID = "claude"
	AgentCursor   AgentID = "cursor"
	AgentWindsurf AgentID = "windsurf"
	AgentDroid    AgentID = "droid"
	AgentGemini   AgentID = "gemini"
)

// ToolKind classifies what kind of action is being gated.
type ToolKind string

const (
	ToolKindShell   ToolKind = "shell"   // terminal / bash command
	ToolKindMCP     ToolKind = "mcp"     // MCP protocol tool
	ToolKindBuiltin ToolKind = "builtin" // agent's built-in tool (Read, Write, etc.)
)

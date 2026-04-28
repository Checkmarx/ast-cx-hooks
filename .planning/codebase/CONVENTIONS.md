# Coding Conventions

**Analysis Date:** 2026-04-28

## Language and Runtime

**Language:** Go 1.21
**Package Management:** Go modules (`go.mod`)

## Naming Patterns

**Files:**
- Use lowercase with no underscores for package directories: `claude/`, `cursor/`, `droid/`, `gemini/`, `windsurf/`
- Test files: `*_test.go` (e.g., `agenthooks_test.go`, `responses_test.go`)
- Package documentation: `doc.go` for package-level comments
- Tests use `_test` suffix on package name: `package agenthooks_test`, `package claude_test`

**Functions:**
- PascalCase for exported functions: `Dispatch()`, `Process()`, `ProcessE()`, `AddRoute()`, `LetStop()`, `ApproveToolUse()`, `SendFollowup()`
- Lowercase for unexported functions: `boolPtr()`, `resolveRouteName()`, `claudeToolKind()`, `cursorPermissionResult()`
- Helper functions use pattern `[VerbAdjective]` for verdict/result helpers: `Resume()`, `Interrupt()`, `Allow()`, `Deny()`, `AcceptWrite()`, `RejectWrite()`

**Variables:**
- PascalCase for exported struct fields: `SessionID`, `WorkDir`, `IsRepeat`, `Agent`, `Proceed`, `ToolName`
- camelCase for local variables: `routes`, `binaryPath`, `home`, `m`, `v`
- Plurals for maps/slices: `routes` (map), `ids` (slice), `installFns` (slice of structs)
- Single-letter iterations acceptable for ranges: `i`, `e`, `k` in loops

**Types:**
- PascalCase for exported types: `RouteFunc`, `AgentID`, `AgentIdleEvent`, `IdleVerdict`, `ToolCallEvent`, `ToolVerdict`, `FileWriteEvent`, `FileWriteVerdict`, `PromptEvent`, `PromptVerdict`
- Struct tags use lowercase with underscores: `json:"session_id"`, `json:"conversation_id"`, `json:"hook_event_name"`
- Abbreviations allowed in type names: `PreToolUseEvent`, `PostToolUseEvent`, `MCPPreEvent`, `PreRunCommandEvent`

**Constants:**
- PascalCase for exported constants: `AgentClaude`, `AgentCursor`, `ToolKindShell`, `ToolKindMCP`
- String values use lowercase: `"claude"`, `"cursor"`, `"shell"`, `"mcp"`

## Package Organization

**Root package (`agenthooks`):**
- Core routing and dispatch logic: `agenthooks.go`
- Unified handler registration and event structures: `unified.go`
- Tests in same package with `_test` suffix: `agenthooks_test.go`

**Platform packages:**
- Each agent has its own package: `claude/`, `cursor/`, `windsurf/`, `droid/`, `gemini/`
- Each platform package contains:
  - `doc.go`: Package documentation with link to official reference
  - `types.go`: Event structures, Result structures, and JSON struct tags
  - `responses.go`: Helper functions for creating result values
  - `responses_test.go`: Tests for helper functions

**Internal packages:**
- Utilities and internal-only code go in `internal/` directory
- Examples: `internal/codec/` (JSON encoding/decoding), `internal/scaffold/` (project initialization)

## Import Organization

**Order (when present):**
1. Standard library imports: `"encoding/json"`, `"os"`, `"fmt"`
2. Blank line
3. Third-party/internal imports: `"github.com/CheckmarxDev/ast-cx-hooks/..."`

**Examples from codebase:**
```go
import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/CheckmarxDev/ast-cx-hooks/internal/codec"
)
```

**No aliases:** Imports use direct package names (e.g., `os.Exit()` not `o.Exit()`)

## Code Style

**Formatting:**
- Standard Go formatting with `gofmt`
- No explicit formatter config detected (uses Go defaults)

**Line length:** Not enforced visibly; code stays reasonable (typical Go convention ~100-120 chars)

**Indentation:** Tabs (Go standard)

**Brackets and spacing:**
- Opening braces on same line: `func main() {`
- Single-line function bodies allowed when simple: `func boolPtr(b bool) *bool { return &b }`
- Spaces around operators: `x := y + z`

## Error Handling

**Pattern 1: Graceful stdin parse failures**
- Bad JSON on stdin causes silent exit with code 0
- This prevents agents from being blocked by payload errors
- Used in `Process()` and `ProcessE()`
```go
if err := codec.DecodeStdin(&in); err != nil {
    os.Exit(0)  // graceful fallthrough
}
```

**Pattern 2: Blocking errors**
- Errors in `ProcessE()` handlers write to stderr and exit code 2
- This signals supporting agents to surface the error and block the action
```go
if err != nil {
    fmt.Fprintln(os.Stderr, err.Error())
    os.Exit(2)  // blocking error
}
```

**Pattern 3: Command errors (install, build)**
- Return wrapped errors with context using `fmt.Errorf()`
- Main handles and prints to stderr
```go
if err := item.fn(home, abs); err != nil {
    fmt.Fprintf(os.Stderr, "warning: %s: %v\n", item.name, err)
}
```

**Pattern 4: Silent JSON unmarshal failures**
- Uses `//nolint:errcheck` on intentional silent failures
- Happens when tool input structure may vary or is optional
- Found in: `unified.go` (lines 480, 614, 654, 668), `cmd/agenthooks/main.go` (line 165)

## Logging and Debugging

**Logging approach:**
- No structured logging framework used
- Uses standard library: `fmt.Printf()`, `fmt.Fprintln()`, `fmt.Fprintf()`
- Stderr reserved for errors and warnings
- Stdout reserved for JSON responses to agents

**stderr usage:**
- Error messages: `fmt.Fprintf(os.Stderr, "...")`
- Status messages during install: `fmt.Printf("✓ %s configured\n", ...)`

## Comments and Documentation

**Pattern 1: Package documentation**
- Every package has `doc.go` with package-level comment
- Includes brief description, usage context, and link to official reference
- Example: `Package claude provides types and response helpers for Claude Code hooks.`

**Pattern 2: Function documentation**
- Public functions have one-line comment describing return/side effect
- Example: `// LetStop returns a decision that allows Claude to stop normally.`
- Private functions rarely documented unless behavior is non-obvious

**Pattern 3: Struct field documentation**
- Usually no field-level comments (names are self-documenting)
- Complex fields have inline comments: `// SessionID is the conversation or session identifier.`

**Pattern 4: Code section separators**
- Uses comment blocks to divide large files into logical sections
- Format: `// =============================================================================`
- Example in `unified.go`:
```go
// =============================================================================
// WhenAgentIdle — unified Stop / AfterAgent / post_cascade_response handler
// =============================================================================
```

**When NOT to comment:**
- Code is already clear from naming
- Obvious implementations like helper functions
- Comments on obvious patterns

## Struct Design

**Event structs:**
- Embedded `*Base` types for shared fields: `EventBase` (Claude/Cursor), common session/work data
- Platform-specific fields follow base fields
- Example from `claude/types.go`:
```go
type StopEvent struct {
    EventBase
    HookActive bool `json:"stop_hook_active"`
}
```

**Result structs:**
- Often use `*Details` pointers for optional platform-specific output
- Keep decision/status in main struct, extra fields in Details struct
- Example:
```go
type PreToolUseResult struct {
    ResultBase
    Details *ToolPermission `json:"hookSpecificOutput,omitempty"`
}
```

## Interface Usage

**Minimal interface usage:**
- No explicit interfaces defined in codebase
- Uses function types for handlers: `type RouteFunc func()`, `type AgentIdleFunc func(AgentIdleEvent) IdleVerdict`
- Function types allow testing via dependency injection

## Generic Usage

**Go generics (1.18+):**
- Used in `Process[I, O any]()` and `ProcessE[I, O any]()` for stdin/stdout codec
- Keeps codec logic generic and reusable across all event/result types
- Not overused; only where it genuinely reduces duplication

## Testing Conventions

**Test file location:** Co-located with implementation (`*_test.go` in same directory)

**Test package naming:** `package <name>_test` to test public API surface

**Test function naming:** `func Test[Feature]()` - describes what is being tested
- `TestAddRouteAndDispatch`
- `TestProcessReadsStdinAndWritesStdout`
- `TestStopResponses`
- `TestPreToolUseResponses`

**Subtable pattern:** Used for testing multiple cases
- Uses `tests` slice of anonymous structs with `name`, `event`, `expect` fields
- Range loop calls `t.Run(tc.name, func(t *testing.T) { ... })`
- Seen in `agenthooks_test.go` for `TestIsLooping`

## Linting and Code Quality

**Linting approach:**
- Uses `//nolint:errcheck` pragmatically for intentional silent JSON unmarshal failures
- No detected linter configuration (likely uses Go defaults)
- Expected conventions follow Go idioms

**Error suppression rationale:**
- JSON unmarshaling can fail silently when input varies across agents
- Tool input structures are heterogeneous and partially optional
- Better to continue with partial data than block the hook

## Pointer Usage

**Rules observed:**
- Pointers to primitives used only when need "nil as missing" semantics
- Example: `Proceed *bool` in results where "not set" is distinct from false
- Fields that might not be present use pointer types: `*Details`, `*SessionStartDetails`
- Required fields use value types: `Decision string` not `*string`

## Constants vs Variables

**Routes map:** Declared as package-level variable to allow mutation via `AddRoute()`
```go
var routes = map[string]RouteFunc{}
```

**String constants for verdict responses:** Used throughout result helpers
```go
const (
    AgentClaude AgentID = "claude"
    // ...
)
```

---

*Convention analysis: 2026-04-28*

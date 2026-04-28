# Architecture

**Analysis Date:** 2026-04-28

## Pattern Overview

**Overall:** Abstraction layer adapter pattern with platform-specific implementations.

The framework provides a **unified hook interface** that abstracts away differences between five AI coding agent platforms (Claude Code, Cursor, Windsurf Cascade, Factory Droid, Gemini CLI). Applications define handlers once using a unified API, and the framework automatically routes platform-specific JSON events through the appropriate adapter.

**Key Characteristics:**
- **Single API, multiple implementations**: Handlers registered via unified functions map to platform-specific routes
- **Event-driven architecture**: Hook binary acts as a passive event processor, invoked by agents
- **Route-based dispatch**: Binary reads `os.Args[1]` to select which handler to invoke
- **Process-based execution**: Each hook invocation is a separate process with stdin/stdout JSON communication
- **Platform-specific flexibility**: Developers can use unified handlers for common use cases or drop to platform-specific code for advanced needs

## Layers

**Framework Core:**
- **Purpose:** Routing and dispatch infrastructure that maps unified handlers to platform-specific routes
- **Location:** `agenthooks.go`, `unified.go`
- **Contains:** Route registry, process control flow, unified handler registration functions
- **Depends on:** Standard library (os, fmt, encoding/json)
- **Used by:** Every hook binary that imports the framework

**Platform Adapters:**
- **Purpose:** Translate platform-specific JSON events to/from unified event types
- **Location:** `claude/`, `cursor/`, `windsurf/`, `droid/`, `gemini/` (types.go, responses.go in each)
- **Contains:** Event type definitions, JSON unmarshaling, response builders
- **Depends on:** Framework core for ProcessE/Process functions
- **Used by:** unified.go for route registration

**Unified Event Abstractions:**
- **Purpose:** Define common interfaces that span all platforms
- **Location:** `unified.go` - AgentIdleEvent, ToolCallEvent, FileWriteEvent, PromptEvent
- **Contains:** Event structs with platform-agnostic fields, verdict types, helper methods
- **Depends on:** Platform packages for Raw field access
- **Used by:** Handler implementations that want one piece of code to work everywhere

**Serialization Layer:**
- **Purpose:** Handle JSON I/O with platform-specific quirks (BOM handling, incomplete writes)
- **Location:** `internal/codec/codec.go`
- **Contains:** DecodeStdin (with UTF-8 BOM stripping), EncodeStdout
- **Depends on:** Standard library (bufio, encoding/json)
- **Used by:** Process and ProcessE for stdin/stdout communication

**Code Generation:**
- **Purpose:** Scaffold starter hook projects to reduce boilerplate
- **Location:** `internal/scaffold/scaffold.go`, `internal/scaffold/templates.go`
- **Contains:** Template file generation for main.go, README, .gitignore, policy.json
- **Depends on:** Nothing (internal to CLI tool)
- **Used by:** `cmd/agenthooks init`

**CLI Tool:**
- **Purpose:** Provide install, init, and build commands for hook projects
- **Location:** `cmd/agenthooks/main.go`
- **Contains:** Route parsing, subcommand handling (init, install, build)
- **Depends on:** scaffold package, exec, filepath, JSON marshaling
- **Used by:** End users building and deploying hook binaries

## Data Flow

**Event Handling Flow:**

1. **Agent invokes hook binary** with route name in argv
   - Example: `myhook claude-pre-tool-use` with JSON event on stdin
2. **Dispatch() routes to handler** based on os.Args[1]
3. **Platform-specific handler wraps user's unified handler**
   - Platform event struct unmarshaled from stdin
   - Unified event built by extracting common fields and setting platform-specific Raw field
4. **User's handler receives unified event** and returns verdict
5. **Platform adapter converts verdict back to platform format** and encodes to stdout
6. **Agent reads JSON response** and acts on it (allow/deny/ask/continue)

**Unified Handler Registration (WhenAgentIdle example):**

1. `WhenAgentIdle(fn)` registers a closure under route "claude-stop"
2. That same function also registers routes: "cursor-stop", "windsurf-post-cascade-response", "droid-stop", "gemini-after-agent"
3. Each route has platform-specific unmarshaling but calls the same `fn` with a unified AgentIdleEvent
4. All five routes exist in the same routes map after registration

**State Management:**
- **No persistent state**: Each hook invocation is stateless (single stdin → stdout process)
- **Session context**: Platform provides SessionID/ConversationID so handler can correlate events within a session
- **IsRepeat/loop detection**: Platforms signal if current hook was triggered by prior hook to prevent loops

## Key Abstractions

**Unified Event Types:**

These represent the four main hook categories abstracted across all platforms:

- `AgentIdleEvent` — Agent finished responding (maps to: Claude Stop, Cursor stop, Windsurf post_cascade_response, Droid Stop, Gemini AfterAgent)
  - Example: `e.IsLooping()` checks AgentClaude/Droid/Gemini `IsRepeat` or Cursor `AutoRetryCount >= 3`
- `ToolCallEvent` — Before tool/command executes (maps to: Claude PreToolUse, Cursor shell+MCP, Windsurf shell+MCP, Droid PreToolUse, Gemini BeforeTool)
  - Example: `e.IsShell()` and `e.IsMCP()` classify the action type
- `FileWriteEvent` — After file is written/edited (maps to: Claude PostToolUse filtered, Cursor afterFileEdit, Windsurf post_write_code, Droid PostToolUse filtered, Gemini AfterTool filtered)
  - Contains: FilePath, Changes (array of before/after diffs)
- `PromptEvent` — User submits prompt before processing (maps to: Claude UserPromptSubmit, Cursor beforeSubmitPrompt, Windsurf pre_user_prompt, Droid UserPromptSubmit, Gemini BeforeAgent)
  - Example: EnrichPrompt appends context, RejectPrompt blocks submission

**Verdict Types:**

Each handler returns a verdict struct with platform-independent fields:

- `IdleVerdict{Proceed bool, Feedback string}` — Resume() or Interrupt()
- `ToolVerdict{Permit bool, Message string, NeedsConfirm bool}` — Allow(), Deny(), AskUser()
- `FileWriteVerdict{Reject bool, Feedback string, Footnote string}` — AcceptWrite(), RejectWrite(), AnnotateWrite()
- `PromptVerdict{Accept bool, Message string}` — AcceptPrompt(), RejectPrompt(), EnrichPrompt()

**Platform-Specific Event Structures:**

Each platform package (claude/, cursor/, etc.) defines its own event types matching agent JSON schemas:

- `claude.PreToolUseEvent` — Claude's tool call event with ToolName, ToolInput (json.RawMessage), ToolUseID
- `cursor.ShellPreEvent` — Cursor's shell hook with Command, WorkDir, Timeout
- `windsurf.PreRunCommandEvent` — Windsurf's shell event with ToolInfo structure
- Similar pattern for all platforms and all event types

**Response Builders:**

Each platform package provides helper functions to construct responses:

- `claude.ApproveToolUse()`, `claude.DenyToolUse(reason)`, `claude.AskUserAboutTool(reason)` — Build PreToolUseResult
- `cursor.Permit()`, `cursor.Forbid(userMsg, agentMsg)`, `cursor.RequestConfirmation(...)` — Build PermissionResult
- Pattern repeats for all platforms and all hook types

## Entry Points

**Hook Binary (User-Facing):**
- **Location:** User's main.go (generated by `agenthooks init` or written manually)
- **Triggers:** Agent invokes the binary with route name in argv, JSON event on stdin
- **Responsibilities:**
  - Register unified handlers (WhenAgentIdle, BeforeToolCall, AfterFileWrite, BeforePrompt) or platform-specific routes (AddRoute)
  - Call Dispatch() at end of main()
  - Handle all event types and return appropriate verdicts

**Scaffold Command (Development):**
- **Location:** `cmd/agenthooks init`
- **Triggers:** User runs `go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks init [--dir <path>]`
- **Responsibilities:**
  - Create directory structure
  - Write main.go template with all four unified handlers wired
  - Write README.md with build/test instructions
  - Write .gitignore and policy.json starter

**Install Command (Deployment):**
- **Location:** `cmd/agenthooks install <binary>`
- **Triggers:** User runs `go run ... install ./myhook`
- **Responsibilities:**
  - Read binary path and resolve to absolute path
  - Locate agent settings files in home directory (~/.claude/settings.json, ~/.cursor/hooks.json, etc.)
  - Inject hook configuration for each agent pointing to the binary
  - Preserve existing agent settings while adding hook entries

**Build Command (Distribution):**
- **Location:** `cmd/agenthooks build`
- **Triggers:** User runs `go run ... build` from hook project root
- **Responsibilities:**
  - Cross-compile the hook binary for all supported platforms (macOS, Linux, Windows × amd64/arm64)
  - Output binaries to dist/ directory with platform-specific names

## Error Handling

**Strategy:** Fail gracefully with minimal impact to agent operations.

**Patterns:**

- **JSON parse errors** (DecodeStdin fails):
  - Process() exits with code 0 (agent assumes hook succeeded)
  - Agent continues without waiting for response (timeout prevents hanging)
  - No error logged to stderr (agent unaware of problem)

- **Handler errors** (explicit return error):
  - ProcessE() used for routes that can block operations (exit 2)
  - Error message written to stderr and seen by agent
  - Platforms that support exit 2 (Windsurf, Droid via exit from ProcessE, Gemini) block the operation
  - Platforms that don't support exit 2 still receive error message in stdout

- **Platform-specific response handling**:
  - Windsurf post_cascade_response is fire-and-forget: Interrupt logged to stderr but ignored by agent
  - Cursor afterFileEdit is fire-and-forget: Reject feedback logged but not sent to agent
  - Platforms with blocking capability receive structured response with decision/reason fields

- **Binary not found**:
  - install command returns error and exits 1, reports to user
  - Agent settings unchanged
  - User can retry or debug path issues

## Cross-Cutting Concerns

**Logging:**
- Approach: None by default. Handlers may use fmt.Fprintf(os.Stderr) or structured logging
- Agents typically capture stderr separately from response JSON
- Recommended: Log errors, not happy path execution

**Validation:**
- Approach: Handlers validate input in user code
- Framework does minimal validation (checks route existence, attempts JSON decode)
- Platform-specific details accessed via Raw field for custom validation

**Authentication:**
- Approach: None. Framework is authentication-agnostic
- Handler responsibility: hook binary has same privileges as agent process
- Security model: hook binary is trusted code (must be under user control)

**Concurrency:**
- Approach: None. Each hook invocation is a separate process
- No shared state between invocations
- Multiple hooks may run in parallel but each has isolated stdin/stdout

**Platform Differences:**

The framework normalizes these differences:

| Aspect | Claude | Cursor | Windsurf | Droid | Gemini |
|--------|--------|--------|----------|-------|--------|
| **Blocking** | JSON response decision field | PermissionResult.permission | Exit code 2 | Exit code 2 | Exit code 2 |
| **Post-hooks** | Can inject feedback | Fire-and-forget | Fire-and-forget | Can inject feedback | Can annotate |
| **User prompts** | UserPromptSubmit | beforeSubmitPrompt | pre_user_prompt | UserPromptSubmit | BeforeAgent |
| **Session ID** | session_id | conversation_id | trajectory_id | session_id | session_id |
| **WorkDir** | cwd field | workspace_roots list | ToolInfo | cwd field | cwd field |
| **MCP tools** | mcp__* prefix | Separate MCPPreEvent | Separate event | mcp__* prefix | Inlined ToolName |

---

*Architecture analysis: 2026-04-28*

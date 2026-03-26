# agenthooks

A Go framework for building hooks that work across **all major AI coding agents** — with a single codebase.

Write one handler, compile one binary, and it works with **Claude Code**, **Cursor**, **Windsurf Cascade**, **Factory Droid**, and **Gemini CLI**.

```go
package main

import "github.com/CheckmarxDev/ast-cx-hooks"

func main() {
    agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
        if e.IsShell() && strings.Contains(e.Command, "rm -rf") {
            return agenthooks.Deny("Destructive commands are not allowed.")
        }
        return agenthooks.Allow()
    })
    agenthooks.Dispatch()
}
```

## Why agenthooks?

Every AI coding agent has its own hook system with different JSON schemas, response formats, and configuration files. `agenthooks` abstracts all of that away:

| Feature | Without agenthooks | With agenthooks |
|---|---|---|
| Hook handlers | 5 separate implementations | 1 unified handler |
| JSON schemas | Learn 5 different formats | Learn 1 event struct |
| Config files | Maintain 5 config files | `agenthooks install` does it |
| Binary builds | Manual per-platform | `agenthooks build` cross-compiles |

## Installation

```bash
go get github.com/CheckmarxDev/ast-cx-hooks
```

## Supported Agents

| Agent | Config Location | Hook Style |
|---|---|---|
| **Claude Code** | `~/.claude/settings.json` | JSON stdin → JSON stdout |
| **Cursor** | `~/.cursor/hooks.json` | JSON stdin → JSON stdout |
| **Windsurf Cascade** | `~/.codeium/windsurf/hooks.json` | JSON stdin, exit code 2 to block |
| **Factory Droid** | `~/.factory/settings.json` | JSON stdin → JSON stdout |
| **Gemini CLI** | `~/.gemini/settings.json` | JSON stdin, exit code 2 to block |

## Unified Hooks

agenthooks provides **4 unified hook categories** that map to platform-specific events automatically:

### `WhenAgentIdle` — Agent finished responding

Fires when the agent completes a response. Use it to force the agent to continue working.

```go
agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
    if e.IsLooping() {
        return agenthooks.Resume() // break infinite loops
    }
    return agenthooks.Interrupt("Please run the tests before finishing.")
})
```

**Platform mapping:**
- Claude Code → `Stop`
- Cursor → `stop`
- Windsurf → `post_cascade_response` *(fire-and-forget)*
- Factory Droid → `Stop`
- Gemini CLI → `AfterAgent`

### `BeforeToolCall` — Gate tool/command execution

Fires before a tool call or shell command executes. Use it to enforce security policies.

```go
agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    if e.IsShell() {
        return agenthooks.AskUser("Please confirm this shell command.")
    }
    if e.IsMCP() {
        return agenthooks.AllowWithNote("MCP tool approved: " + e.ToolName)
    }
    return agenthooks.Allow()
})
```

**Verdicts:** `Allow()`, `AllowWithNote(msg)`, `Deny(reason)`, `AskUser(reason)`

**Platform mapping:**
- Claude Code → `PreToolUse`
- Cursor → `beforeShellExecution` + `beforeMCPExecution`
- Windsurf → `pre_run_command` + `pre_mcp_tool_use`
- Factory Droid → `PreToolUse`
- Gemini CLI → `BeforeTool`

### `AfterFileWrite` — React to file edits

Fires after the agent writes or edits a file. Use it to run linters, scanners, or inject feedback.

```go
agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
    if strings.HasSuffix(e.FilePath, ".go") {
        // Run a linter, scanner, etc.
        return agenthooks.AnnotateWrite("Reminder: run go vet before committing.")
    }
    return agenthooks.AcceptWrite()
})
```

**Verdicts:** `AcceptWrite()`, `RejectWrite(reason)`, `AnnotateWrite(note)`

**Platform mapping:**
- Claude Code → `PostToolUse` (Write/Edit tools)
- Cursor → `afterFileEdit`
- Windsurf → `post_write_code`
- Factory Droid → `PostToolUse` (Write/Edit tools)
- Gemini CLI → `AfterTool` (Write/Edit tools)

### `BeforePrompt` — Filter or enrich user prompts

Fires when the user submits a prompt, before the agent processes it.

```go
agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
    if containsSensitiveInfo(e.Text) {
        return agenthooks.RejectPrompt("Prompt contains sensitive information.")
    }
    return agenthooks.EnrichPrompt("Always follow our coding standards.")
})
```

**Verdicts:** `AcceptPrompt()`, `RejectPrompt(msg)`, `EnrichPrompt(context)`

**Platform mapping:**
- Claude Code → `UserPromptSubmit`
- Cursor → `beforeSubmitPrompt`
- Windsurf → `pre_user_prompt`
- Factory Droid → `UserPromptSubmit`
- Gemini CLI → `BeforeAgent`

## Platform-Specific Hooks

For advanced use cases that need platform-specific event data, use `AddRoute` with per-platform packages:

```go
import (
    "github.com/CheckmarxDev/ast-cx-hooks"
    "github.com/CheckmarxDev/ast-cx-hooks/claude"
)

agenthooks.AddRoute("claude-pre-tool-use", func() {
    agenthooks.Process(func(e claude.PreToolUseEvent) claude.PreToolUseResult {
        if e.ToolName == "Bash" {
            return claude.DenyToolUse("Shell commands are disabled.")
        }
        return claude.ApproveToolUse()
    })
})
```

### Available packages

| Package | Import Path |
|---|---|
| Claude Code | `github.com/CheckmarxDev/ast-cx-hooks/claude` |
| Cursor | `github.com/CheckmarxDev/ast-cx-hooks/cursor` |
| Windsurf | `github.com/CheckmarxDev/ast-cx-hooks/windsurf` |
| Factory Droid | `github.com/CheckmarxDev/ast-cx-hooks/droid` |
| Gemini CLI | `github.com/CheckmarxDev/ast-cx-hooks/gemini` |

## Building & Installing

### Scaffold a new hooks project

```bash
# Create a starter hooks project in the current directory
go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks init

# Or scaffold into a specific directory
go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks init --dir ./my-hooks-project
```

The scaffold command generates:
- `main.go` with all 4 unified hook handlers wired
- `README.md` with quick-start and local testing commands
- `.gitignore` for common build outputs and secrets
- `policy.json` starter policy you can customize

### Build your hook binary

```bash
# Build for current platform
go build -o myhook .

# Cross-compile for all supported platforms (macOS, Linux, Windows × amd64/arm64)
go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks build
# Outputs to dist/
```

### Install hooks into all agents

```bash
go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks install ./myhook
```

This automatically writes the correct configuration into each agent's settings file:
- `~/.claude/settings.json`
- `~/.cursor/hooks.json`
- `~/.codeium/windsurf/hooks.json`
- `~/.factory/settings.json`

## Complete Example

Here's a complete hook binary that works across all 5 agents:

```go
package main

import (
    "strings"

    "github.com/CheckmarxDev/ast-cx-hooks"
)

func main() {
    // Gate dangerous shell commands
    agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
        if e.IsShell() {
            for _, banned := range []string{"rm -rf", "DROP TABLE", "format"} {
                if strings.Contains(e.Command, banned) {
                    return agenthooks.Deny("Blocked: command contains '" + banned + "'")
                }
            }
        }
        return agenthooks.Allow()
    })

    // Force tests before agent finishes
    agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
        if e.IsLooping() {
            return agenthooks.Resume()
        }
        return agenthooks.Interrupt("Please run all tests before finishing.")
    })

    // Block prompts with secrets
    agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
        if strings.Contains(e.Text, "API_KEY") {
            return agenthooks.RejectPrompt("Do not include API keys in prompts.")
        }
        return agenthooks.AcceptPrompt()
    })

    agenthooks.Dispatch()
}
```

### Build & install it:

```bash
go build -o myhook .
go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks install ./myhook
```

## Testing with Different Agents

### Manual Testing (any agent)

Hook binaries read JSON from stdin and write JSON to stdout. You can test them directly:

```bash
# Build your hook
go build -o myhook .

# Test the "before tool call" hook (Claude Code format)
echo '{"tool_name":"Bash","tool_input":{"command":"rm -rf /"},"session_id":"test","cwd":"/tmp"}' | ./myhook claude-pre-tool-use

# Test the "agent idle" hook (Cursor format)
echo '{"status":"completed","loop_count":0,"conversation_id":"test"}' | ./myhook cursor-stop

# Test a Gemini CLI hook
echo '{"tool_name":"shell","tool_input":{"command":"ls"},"session_id":"test","cwd":"/tmp"}' | ./myhook gemini-before-tool
```

The first argument is the **route name** — it tells the binary which handler to invoke.

### Testing with Claude Code

1. Build and install:
   ```bash
   go build -o myhook .
   go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks install ./myhook
   ```
2. Open Claude Code — your hooks are now active.
3. Try triggering a hooked action (e.g., ask Claude to run a shell command).
4. Check `~/.claude/settings.json` to see the generated config.

### Testing with Cursor

1. Build and install (same as above).
2. Open Cursor IDE — hooks are configured in `~/.cursor/hooks.json`.
3. Use the agent to run shell commands or edit files to trigger hooks.

### Testing with Windsurf Cascade

1. Build and install.
2. Open Windsurf — hooks are in `~/.codeium/windsurf/hooks.json`.
3. Note: Windsurf pre-hooks block via **exit code 2**; post-hooks are fire-and-forget.

### Testing with Factory Droid

1. Build and install.
2. Open Factory — hooks are in `~/.factory/settings.json`.
3. Droid hooks follow the same stdin/stdout JSON pattern as Claude.

### Testing with Gemini CLI

1. Build and install.
2. Currently the `install` command does not auto-configure Gemini CLI. Manually add hooks to `~/.gemini/settings.json`:
   ```json
   {
     "hooks": {
       "BeforeTool": [{ "command": "/path/to/myhook gemini-before-tool" }],
       "AfterAgent": [{ "command": "/path/to/myhook gemini-after-agent" }],
       "BeforeAgent": [{ "command": "/path/to/myhook gemini-before-agent" }],
       "AfterTool": [{ "command": "/path/to/myhook gemini-after-file-tool" }]
     }
   }
   ```
3. Run `gemini` — your hooks are active.

### Unit Testing Your Hooks

Write standard Go tests for your handler logic:

```go
func TestDenyDangerousCommands(t *testing.T) {
    event := agenthooks.ToolCallEvent{
        Kind:    agenthooks.ToolKindShell,
        Command: "rm -rf /important",
    }
    verdict := myToolCallHandler(event)
    if verdict.Permit {
        t.Fatal("expected dangerous command to be denied")
    }
}
```

## Architecture

```
github.com/CheckmarxDev/ast-cx-hooks
├── agenthooks.go      # Core API: AddRoute, Dispatch, Process, ProcessE
├── unified.go         # Unified hooks: WhenAgentIdle, BeforeToolCall, etc.
├── claude/            # Claude Code types, events & response helpers
├── cursor/            # Cursor IDE types, events & response helpers
├── windsurf/          # Windsurf Cascade types, events & response helpers
├── droid/             # Factory Droid types, events & response helpers
├── gemini/            # Gemini CLI types, events & response helpers
├── internal/codec/    # JSON stdin/stdout serialization
├── internal/scaffold/ # Templates and generator for `agenthooks init`
└── cmd/agenthooks/    # CLI tool: init, install, and build commands
```

### How it works

1. Your `main()` registers unified handlers (e.g., `BeforeToolCall`).
2. Each unified handler internally registers platform-specific routes.
3. `Dispatch()` reads `os.Args[1]` to select the correct route.
4. `Process()` reads JSON from stdin, calls your handler, writes JSON to stdout.
5. The agent invokes your binary with a subcommand like `myhook claude-pre-tool-use`.

## API Reference

### Core Functions

| Function | Description |
|---|---|
| `AddRoute(name, fn)` | Register a handler for a specific route name |
| `Dispatch()` | Run the matching handler based on `os.Args[1]` |
| `Process(handler)` | Read JSON stdin → call handler → write JSON stdout |
| `ProcessE(handler)` | Like `Process` but supports blocking errors (exit 2) |

### Unified Hooks

| Function | When it fires |
|---|---|
| `WhenAgentIdle(fn)` | Agent finishes responding |
| `BeforeToolCall(fn)` | Before a tool/command executes |
| `AfterFileWrite(fn)` | After a file is written/edited |
| `BeforePrompt(fn)` | Before a user prompt is processed |

### Event Helpers

| Method | On | Description |
|---|---|---|
| `e.IsLooping()` | `AgentIdleEvent` | Detects infinite loops |
| `e.IsShell()` | `ToolCallEvent` | Is this a shell command? |
| `e.IsMCP()` | `ToolCallEvent` | Is this an MCP tool call? |



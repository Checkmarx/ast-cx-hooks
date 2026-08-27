# Usage Guide

This guide covers how to use **cxagenthooks** (`github.com/Checkmarx/ast-cx-hooks`) —
one hook codebase that runs across every supported AI coding agent. You write a single
Go handler, compile one binary, and the library translates the wire format (JSON schema,
response shape, blocking semantics) for each platform.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Unified Hooks](#unified-hooks)
  - [WhenAgentIdle](#whenagentidle)
  - [BeforeToolCall](#beforetoolcall)
  - [BeforeFileEdit](#beforefileedit)
  - [AfterFileWrite](#afterfilewrite)
  - [BeforePrompt](#beforeprompt)
  - [WhenSubagentIdle](#whensubagentidle)
  - [AfterToolFailure](#aftertoolfailure)
  - [BeforeFileRead](#beforefileread)
- [Verdicts Reference](#verdicts-reference)
- [Scenarios (multiple implementations per hook)](#scenarios-multiple-implementations-per-hook)
- [Platform-Specific Hooks](#platform-specific-hooks)
- [CLI Reference](#cli-reference)
- [Testing Hooks Locally](#testing-hooks-locally)
- [Example Workflows](#example-workflows)

---

## Overview

An **agent hook** is a small program the AI coding agent runs at defined moments — before
it executes a tool, before it writes a file, when it finishes responding, and so on. Each
agent has its own hook protocol (config location, JSON schema, response shape, blocking
semantics). `cxagenthooks` gives you **one** API: you register handlers for **8 unified
hook categories**, and the library maps them to each platform's native events.

**Supported agents:**

| Agent | Config location | Output style |
|---|---|---|
| Claude Code | `~/.claude/settings.json` | nested JSON decision |
| Cursor | `~/.cursor/hooks.json` | flat JSON decision |
| Windsurf Cascade | `~/.codeium/windsurf/hooks.json` | exit code `2` only |
| Factory Droid | `~/.factory/settings.json` | nested JSON / exit `2` |
| Gemini CLI | `~/.gemini/settings.json` | JSON decision / exit `2` |
| GitHub Copilot (VS Code) | `.github/hooks/*.json` (project-scoped) | nested JSON decision |
| GitHub Copilot CLI | `~/.copilot/hooks/agenthooks.json` | flat JSON / exit `2` |
| OpenAI Codex CLI | `~/.codex/hooks.json` | nested JSON decision |

**How it runs:** the agent invokes your compiled binary with a **route** as the first
argument (e.g. `myhook claude-pre-tool-use`). The binary reads the event JSON from
**stdin**, runs your handler, and writes the platform-specific response to **stdout** (or
signals a block via exit code `2`, depending on the agent).

---

## Prerequisites

Before building hooks you need:

1. **Go 1.23 or later** installed (`go version`).
2. An initialized Go module for your hooks project (`go mod init …`).
3. At least one supported agent installed and configured on your machine.

The library has **zero third-party dependencies** — standard library only.

---

## Quick Start

The fastest path is the `agenthooks` CLI, which scaffolds, builds, and installs for you.

```bash
# 1. Scaffold a starter project (main.go + policy.json + README + .gitignore)
go run github.com/Checkmarx/ast-cx-hooks/cmd/agenthooks init --dir ./my-hooks
cd ./my-hooks

# 2. Initialize your module and pull the dependency
go mod init github.com/your-org/my-hooks
go get github.com/Checkmarx/ast-cx-hooks@latest
go mod tidy

# 3. Build your hook binary
go build -o my-hooks .

# 4. Write the correct hook config — in each agent's own shape — into every settings file
go run github.com/Checkmarx/ast-cx-hooks/cmd/agenthooks install ./my-hooks
```

A minimal `main.go` looks like this:

```go
package main

import (
    "strings"

    "github.com/Checkmarx/ast-cx-hooks"
)

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

`Dispatch()` inspects `os.Args[1]` (the route the agent passed), decodes stdin into the
matching event, calls your handler, and writes the response. Register only the hooks you
need; everything else is a no-op.

---

## Unified Hooks

Register a default handler for any of the 8 categories. Each maps automatically to the
native event(s) of every platform that supports it (see the
[support matrix in the README](../README.md#support-matrix) for coverage per agent).

### `WhenAgentIdle`

Fires when the agent finishes responding. Use it to keep the agent working until a
condition is met, or to break infinite continuation loops.

```go
agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
    if e.IsLooping() {
        return agenthooks.Resume() // break infinite continuation loops
    }
    return agenthooks.Interrupt("Please run the tests before finishing.")
})
```

Verdicts: `Resume()` · `Interrupt(feedback)`

### `BeforeToolCall`

Gate tool and command execution before it runs.

```go
agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    if e.IsShell() {
        return agenthooks.AskUser("Please confirm this shell command.")
    }
    return agenthooks.AllowWithInput(sanitize(e.ToolArgs)) // rewrite tool input before it runs
})
```

Event helpers: `e.IsShell()` · `e.IsMCP()`

Verdicts: `Allow()` · `AllowWithNote(msg)` · `AllowWithContext(ctx)` · `AllowWithInput(json)` · `AskUser(reason)` · `Deny(reason)` · `DenyWithContext(reason, ctx)`

### `BeforeFileEdit`

A pre-write **gate** — fires before the agent writes/edits a file, with the proposed
`Changes` available, so a handler can **deny** the write before it lands. This is the
route a security scan uses to block a vulnerable change before it reaches disk.

```go
agenthooks.BeforeFileEdit(func(e agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
    for _, d := range e.Changes {
        if scanForSecret(d.After) {
            return agenthooks.RejectEditWithContext(
                "Secret detected in "+e.FilePath,
                "Remove the secret and retry the write.") // additionalContext steers the agent
        }
    }
    return agenthooks.AcceptEdit()
})
```

Verdicts: `AcceptEdit()` · `AcceptEditWithInput(json)` · `RejectEdit(reason)` · `RejectEditWithContext(reason, ctx)` · `AskBeforeEdit(reason)`

### `AfterFileWrite`

React to a completed file edit. Fires post-write, so it can annotate but not block.

```go
agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
    if strings.HasSuffix(e.FilePath, ".go") {
        return agenthooks.AnnotateWrite("Reminder: run `go vet` before committing.")
    }
    return agenthooks.AcceptWrite()
})
```

Verdicts: `AcceptWrite()` · `RejectWrite(reason)` · `RejectWriteWithContext(reason, ctx)` · `AnnotateWrite(note)`

### `BeforePrompt`

Filter or enrich user prompts before the agent sees them.

```go
agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
    if containsSecret(e.Text) {
        return agenthooks.RejectPrompt("Prompt contains sensitive information.")
    }
    return agenthooks.EnrichPrompt("Always follow our coding standards.")
})
```

Verdicts: `AcceptPrompt()` · `RejectPrompt(msg)` · `EnrichPrompt(context)`

### `WhenSubagentIdle`

Gate subagent completion. Reuses `AgentIdleEvent` / `IdleVerdict`; `Interrupt` blocks the
subagent from stopping.

```go
agenthooks.WhenSubagentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
    return agenthooks.Resume()
})
```

Verdicts: `Resume()` · `Interrupt(feedback)`

### `AfterToolFailure`

React to a failed tool call.

```go
agenthooks.AfterToolFailure(func(e agenthooks.ToolFailureEvent) agenthooks.ToolFailureVerdict {
    return agenthooks.AnnotateFailure("Tool " + e.ToolName + " failed: " + e.Error)
})
```

Verdicts: `AcknowledgeFailure()` · `AnnotateFailure(note)` · `RejectAfterFailure(reason)`

### `BeforeFileRead`

Gate the agent reading a file.

```go
agenthooks.BeforeFileRead(func(e agenthooks.FileReadEvent) agenthooks.FileReadVerdict {
    if strings.HasSuffix(e.FilePath, ".env") {
        return agenthooks.DenyRead("Reading secret files is not allowed.")
    }
    return agenthooks.AllowRead()
})
```

Verdicts: `AllowRead()` · `DenyRead(reason)`

---

## Verdicts Reference

Each hook returns a typed verdict. Where a verdict carries an `additionalContext` /
`WithContext` variant, that context steers the agent — but it is only delivered where the
agent's hook protocol defines the field (Claude and VS Code Copilot on the pre-tool /
pre-file gates); elsewhere the deny carries its reason only and the context is dropped and
logged. See the [support matrix](../README.md#support-matrix) for the exact coverage.

| Hook | Verdict constructors |
|---|---|
| `WhenAgentIdle` / `WhenSubagentIdle` | `Resume()` · `Interrupt(feedback)` |
| `BeforeToolCall` | `Allow()` · `AllowWithNote(msg)` · `AllowWithContext(ctx)` · `AllowWithInput(json)` · `AskUser(reason)` · `Deny(reason)` · `DenyWithContext(reason, ctx)` |
| `BeforeFileEdit` | `AcceptEdit()` · `AcceptEditWithInput(json)` · `RejectEdit(reason)` · `RejectEditWithContext(reason, ctx)` · `AskBeforeEdit(reason)` |
| `AfterFileWrite` | `AcceptWrite()` · `RejectWrite(reason)` · `RejectWriteWithContext(reason, ctx)` · `AnnotateWrite(note)` |
| `BeforePrompt` | `AcceptPrompt()` · `RejectPrompt(msg)` · `EnrichPrompt(context)` |
| `AfterToolFailure` | `AcknowledgeFailure()` · `AnnotateFailure(note)` · `RejectAfterFailure(reason)` |
| `BeforeFileRead` | `AllowRead()` · `DenyRead(reason)` |

---

## Scenarios (multiple implementations per hook)

Several teams — or scenarios — can share **one** hook binary. Register the **default**
with `BeforeToolCall(fn)` and any number of **named scenarios** with
`BeforeToolCallScenario(name, fn)`. Every unified hook has a matching `…Scenario` variant.

```go
agenthooks.BeforeToolCallScenario("phoenix", phoenix.Handle)
agenthooks.BeforeToolCallScenario("cypher",  cypher.Handle)
agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    return agenthooks.Allow() // default when no scenario key matches
})
```

The scenario key is supplied per invocation (default selector = `ScenarioFromArg`):

| Channel | How |
|---|---|
| CLI arg | `myhook claude-pre-tool-use phoenix` |
| Env var | `AGENTHOOKS_SCENARIO=phoenix myhook claude-pre-tool-use` |
| Event content | `agenthooks.UseScenarioSelector(func(m agenthooks.ScenarioMeta) string { … })` |

Resolution order: a matching named scenario → the registered default → a fallback verdict.
For the gating hooks (`BeforeToolCall`, `BeforeFileEdit`, `BeforeFileRead`, `BeforePrompt`)
the fallback is **fail-closed** — a misconfigured scenario with no match **denies** rather
than allowing, and logs why.

---

## Platform-Specific Hooks

Need raw, per-platform event data instead of the unified vocabulary? Use `AddRoute` with a
platform package:

```go
import (
    "github.com/Checkmarx/ast-cx-hooks"
    "github.com/Checkmarx/ast-cx-hooks/claude"
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

Platform packages: `claude` · `cursor` · `windsurf` · `droid` · `gemini` · `copilot` ·
`copilotcli` · `codex`. Each models its agent's full event surface and ships response
builders (`additionalContext`, tool-input/output rewrite, permission decisions, and more).

`Process` reads stdin, runs the handler, and writes stdout; a stdin parse error exits `0`
so a bad payload never blocks the agent. `ProcessE` is the same but lets the handler return
an error to block via exit code `2`.

---

## CLI Reference

Run the CLI with `go run github.com/Checkmarx/ast-cx-hooks/cmd/agenthooks <command>`
(or build it once and put `agenthooks` on your `PATH`).

### `init`

```bash
agenthooks init [--dir <path>]   # default: current directory
```

Scaffolds a starter project: `main.go`, `policy.json`, `README.md`, and `.gitignore`. It
refuses to overwrite any existing file.

### `build`

```bash
agenthooks build [package]       # default package: .
```

Cross-compiles your hook binary into `dist/` for macOS, Linux, and Windows on both `amd64`
and `arm64` (`CGO_ENABLED=0`). For a single local build, plain `go build -o myhook .` works.

### `install`

```bash
agenthooks install <binary-path>
```

Writes the correct hook config — in each agent's own shape — into every settings file:
`~/.claude/settings.json`, `~/.cursor/hooks.json`, `~/.codeium/windsurf/hooks.json`,
`~/.factory/settings.json`, `~/.gemini/settings.json`, `~/.copilot/hooks/agenthooks.json`,
and `~/.codex/hooks.json`.

`install` is **non-destructive**: it merges into existing settings (preserving your other
hooks), aborts rather than overwriting a file it cannot parse, and writes a `.bak` backup
first. The VS Code Copilot extension is **project-scoped** (`.github/hooks/*.json`) and is
set up by hand — see the [README](../README.md#cli-scaffold-build-install).

**Embedding installation.** Consumers that wire these hooks into their own CLI can import
the `install` package directly (`install.InstallClaude/InstallCursor/…`, `install.FormatCommand`,
`install.CmdForFunc`) to control the command each route maps to (e.g. `cx hooks <route>`).

---

## Testing Hooks Locally

Hook binaries read JSON from stdin and write JSON to stdout; the first arg is the
**route**. Pipe a synthetic event in to exercise a handler end-to-end:

```bash
go build -o myhook .

# Claude pre-file-write (block a vulnerable write before it lands)
echo '{"tool_name":"Write","tool_input":{"file_path":"/app/x.py","content":"eval(user)"},"session_id":"t"}' | ./myhook claude-pre-file-write

# Cursor stop
echo '{"status":"completed","loop_count":0,"conversation_id":"t"}' | ./myhook cursor-stop

# Copilot CLI pre-tool-use (lowercase tool names + FLAT output)
echo '{"hook_event_name":"PreToolUse","tool_name":"bash","tool_input":{"command":"rm -rf /"}}' | ./myhook copilot-cli-pre-tool-use

# Codex CLI pre-tool-use (Claude-style nested output)
echo '{"session_id":"t","tool_name":"Bash","tool_input":{"command":"rm -rf /"}}' | ./myhook codex-pre-tool-use
```

Because handlers are plain Go, you can also unit-test them directly — no I/O needed:

```go
func TestDenyDangerousCommands(t *testing.T) {
    e := agenthooks.ToolCallEvent{Kind: agenthooks.ToolKindShell, Command: "rm -rf /important"}
    if myToolCallHandler(e).Permit {
        t.Fatal("expected dangerous command to be denied")
    }
}
```

---

## Example Workflows

### Block secrets before they reach disk

```
1. agenthooks init --dir ./sec-hooks         → scaffold a project
2. Implement BeforeFileEdit to scan Changes  → RejectEditWithContext on a hit
3. go build -o sec-hooks .                    → compile one binary
4. agenthooks install ./sec-hooks             → wire into every agent
```

### Gate destructive shell commands across all agents

```
1. Register BeforeToolCall                    → Deny when e.IsShell() && dangerous
2. echo '{...}' | ./myhook cursor-before-shell  → verify locally
3. agenthooks install ./myhook                → deploy the same gate everywhere
```

### Keep the agent working until tests pass

```
1. Register WhenAgentIdle                     → Interrupt("run the tests") until green
2. Guard with e.IsLooping() → Resume()        → avoid infinite continuation loops
```

---

For architecture, the full support matrix, and the security/fail-open-vs-fail-closed model,
see the [README](../README.md). For contribution setup see
[CONTRIBUTING.md](../CONTRIBUTING.md), and for common problems see
[troubleshooting.md](troubleshooting.md).

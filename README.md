# Checkmarx Agent Hooks

[![CI](https://github.com/Checkmarx/ast-cx-hooks/actions/workflows/ci.yml/badge.svg)](https://github.com/Checkmarx/ast-cx-hooks/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Checkmarx/ast-cx-hooks.svg)](https://pkg.go.dev/github.com/Checkmarx/ast-cx-hooks)
[![Go Report Card](https://goreportcard.com/badge/github.com/Checkmarx/ast-cx-hooks)](https://goreportcard.com/report/github.com/Checkmarx/ast-cx-hooks)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Checkmarx/ast-cx-hooks)](go.mod)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE.txt)
[![Release](https://img.shields.io/github/v/release/Checkmarx/ast-cx-hooks)](https://github.com/Checkmarx/ast-cx-hooks/releases)

One hook codebase for every AI coding agent. Write a single handler, compile one
binary, and it runs across **Claude Code · Cursor · Windsurf Cascade · Factory Droid ·
Gemini CLI · GitHub Copilot (VS Code) · GitHub Copilot CLI** — no per-platform code.

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

That one handler gates shell commands in all seven agents — the library translates the
wire format (JSON schema, response shape, blocking semantics) per platform.

## Installation

```bash
go get github.com/Checkmarx/ast-cx-hooks
```

Zero dependencies (standard library only).

## Supported agents

| Agent | Config location | Output style |
|---|---|---|
| Claude Code | `~/.claude/settings.json` | nested JSON decision |
| Cursor | `~/.cursor/hooks.json` | flat JSON decision |
| Windsurf Cascade | `~/.codeium/windsurf/hooks.json` | exit code `2` only |
| Factory Droid | `~/.factory/settings.json` | nested JSON / exit `2` |
| Gemini CLI | `~/.gemini/settings.json` | JSON decision / exit `2` |
| GitHub Copilot (VS Code) | `.github/hooks/*.json` (project-scoped) | nested JSON decision |
| GitHub Copilot CLI | `~/.copilot/hooks/agenthooks.json` | flat JSON / exit `2` |

The VS Code Copilot extension and the Copilot **CLI** are different products with
incompatible hook schemas (nested vs. flat output, `updatedInput` vs. `modifiedArgs`,
`runTerminalCommand` vs. `bash`), so they are separate packages — `copilot` and `copilotcli`.

## Unified hooks

**8 unified hook categories** map automatically to each platform's native events.

### `WhenAgentIdle` — the agent finished responding
```go
agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
    if e.IsLooping() {
        return agenthooks.Resume() // break infinite continuation loops
    }
    return agenthooks.Interrupt("Please run the tests before finishing.")
})
```
Verdicts: `Resume()` · `Interrupt(feedback)`

### `BeforeToolCall` — gate tool & command execution
```go
agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    if e.IsShell() {
        return agenthooks.AskUser("Please confirm this shell command.")
    }
    return agenthooks.AllowWithInput(sanitize(e.ToolArgs)) // rewrite tool input before it runs
})
```
Verdicts: `Allow()` · `AllowWithNote(msg)` · `AllowWithContext(ctx)` · `AllowWithInput(json)` · `AskUser(reason)` · `Deny(reason)` · `DenyWithContext(reason, ctx)`

### `BeforeFileEdit` — gate a file write **before** it lands (can block)
Fires before the agent writes/edits a file, with the proposed `Changes` available so a
handler can scan the content and **deny** the write — the route a security scan uses to
block a vulnerable change before it reaches disk. (Contrast with `AfterFileWrite`, which
fires post-write and can only annotate.)
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

### `AfterFileWrite` — react to a completed file edit
```go
agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
    if strings.HasSuffix(e.FilePath, ".go") {
        return agenthooks.AnnotateWrite("Reminder: run `go vet` before committing.")
    }
    return agenthooks.AcceptWrite()
})
```
Verdicts: `AcceptWrite()` · `RejectWrite(reason)` · `RejectWriteWithContext(reason, ctx)` · `AnnotateWrite(note)`

### `BeforePrompt` — filter or enrich user prompts
```go
agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
    if containsSecret(e.Text) {
        return agenthooks.RejectPrompt("Prompt contains sensitive information.")
    }
    return agenthooks.EnrichPrompt("Always follow our coding standards.")
})
```
Verdicts: `AcceptPrompt()` · `RejectPrompt(msg)` · `EnrichPrompt(context)`

### `WhenSubagentIdle` — gate subagent completion
Reuses `AgentIdleEvent` / `IdleVerdict`; `Interrupt` blocks the subagent from stopping.
Verdicts: `Resume()` · `Interrupt(feedback)`

### `AfterToolFailure` — react to failed tool calls
```go
agenthooks.AfterToolFailure(func(e agenthooks.ToolFailureEvent) agenthooks.ToolFailureVerdict {
    return agenthooks.AnnotateFailure("Tool " + e.ToolName + " failed: " + e.Error)
})
```
Verdicts: `AcknowledgeFailure()` · `AnnotateFailure(note)` · `RejectAfterFailure(reason)`

### `BeforeFileRead` — gate the agent reading a file
```go
agenthooks.BeforeFileRead(func(e agenthooks.FileReadEvent) agenthooks.FileReadVerdict {
    if strings.HasSuffix(e.FilePath, ".env") {
        return agenthooks.DenyRead("Reading secret files is not allowed.")
    }
    return agenthooks.AllowRead()
})
```
Verdicts: `AllowRead()` · `DenyRead(reason)`

## Support matrix

Which unified hooks each agent supports today:

| Unified hook | Claude | Cursor | Windsurf | Droid | Gemini | Copilot | Copilot CLI |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `WhenAgentIdle` | ✅ | ✅ | ✅ ¹ | ✅ | ✅ | ✅ | ✅ |
| `BeforeToolCall` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `BeforeFileEdit` | ✅ ⁴ | ✅ | ✅ | ✅ | ✅ | ✅ ⁴ | ✅ |
| `AfterFileWrite` | ✅ | ✅ | ✅ ¹ | ✅ | ✅ | ✅ | ✅ |
| `BeforePrompt` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ ³ |
| `WhenSubagentIdle` | ✅ | ✅ | — | ✅ | — | ✅ | ✅ |
| `AfterToolFailure` | ✅ | ✅ ² | — | — | — | — | ✅ |
| `BeforeFileRead` | — | ✅ | ✅ | — | — | — | — |

<sub>¹ fire-and-forget — feedback is logged, not enforced by the agent.  ² observational on Cursor — the verdict is ignored.  ³ observational on Copilot CLI — `userPromptSubmitted` output is not processed.  ⁴ `additionalContext` on a deny is delivered only where the agent's hook protocol defines that field — Claude and VS Code Copilot on the pre-tool / pre-file gates; elsewhere a deny carries its reason only (context is dropped and logged).</sub>

## Multiple implementations per hook (scenarios)

Several teams — or scenarios — can share **one** hook. Register the **default** with
`BeforeToolCall(fn)` and any number of **named scenarios** with `BeforeToolCallScenario(name, fn)`.
The caller passes a **scenario key**; the framework runs the matching handler (or the default).

```go
agenthooks.BeforeToolCallScenario("phoenix", phoenix.Handle)
agenthooks.BeforeToolCallScenario("cypher",  cypher.Handle)
agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    return agenthooks.Allow() // default when no scenario key matches
})
```

The key is supplied per invocation (default selector = `ScenarioFromArg`):

| Channel | How |
|---|---|
| CLI arg | `myhook claude-pre-tool-use phoenix` |
| Env var | `AGENTHOOKS_SCENARIO=phoenix myhook claude-pre-tool-use` |
| Event content | `agenthooks.UseScenarioSelector(func(m agenthooks.ScenarioMeta) string { … })` |

Resolution order: a matching named scenario → the registered default → a fallback verdict
(fail-**closed** for the gating hooks). Every unified hook has a `…Scenario` variant.

## CLI: scaffold, build, install

```bash
# Scaffold a starter hooks project (main.go + policy.json + .gitignore)
go run github.com/Checkmarx/ast-cx-hooks/cmd/agenthooks init --dir ./my-hooks

# Build for the current platform, or cross-compile to dist/ (macOS/Linux/Windows × amd64/arm64)
go build -o myhook .
go run github.com/Checkmarx/ast-cx-hooks/cmd/agenthooks build

# Write the correct hook config — in each agent's own shape — into every settings file
go run github.com/Checkmarx/ast-cx-hooks/cmd/agenthooks install ./myhook
```

`install` writes to `~/.claude/settings.json`, `~/.cursor/hooks.json`,
`~/.codeium/windsurf/hooks.json`, `~/.factory/settings.json`, `~/.gemini/settings.json`,
and `~/.copilot/hooks/agenthooks.json`. The VS Code Copilot extension is project-scoped
(`.github/hooks/*.json`) and is set up by hand.

**Embedding installation.** Consumers that wire these hooks into their own CLI can import the
`install` package directly — `install.InstallClaude/InstallCursor/InstallWindsurf/InstallDroid/InstallGemini/InstallCopilotCLI(home, cmdFor)`
plus `install.FormatCommand` and the `install.CmdForFunc` type — to control the command each
route maps to (e.g. `cx hooks <route>`). Route → settings-file / event-key / encoding is driven by
the single `agenthooks.Catalog` source of truth.

## Platform-specific hooks

Need raw, per-platform event data? Use `AddRoute` with a platform package:

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

Packages: `claude` · `cursor` · `windsurf` · `droid` · `gemini` · `copilot` · `copilotcli`.
Each models its agent's full event surface and ships response builders (`additionalContext`,
tool-input/output rewrite, permission decisions, and more).

## Testing

Hook binaries read JSON from stdin and write JSON to stdout; the first arg is the **route**:

```bash
go build -o myhook .

# Claude pre-file-write (block a vulnerable write before it lands)
echo '{"tool_name":"Write","tool_input":{"file_path":"/app/x.py","content":"eval(user)"},"session_id":"t"}' | ./myhook claude-pre-file-write

# Cursor stop
echo '{"status":"completed","loop_count":0,"conversation_id":"t"}' | ./myhook cursor-stop

# Copilot CLI pre-tool-use (lowercase tool names + FLAT output)
echo '{"hook_event_name":"PreToolUse","tool_name":"bash","tool_input":{"command":"rm -rf /"}}' | ./myhook copilot-cli-pre-tool-use
```

Unit-test a handler directly — it's plain Go:

```go
func TestDenyDangerousCommands(t *testing.T) {
    e := agenthooks.ToolCallEvent{Kind: agenthooks.ToolKindShell, Command: "rm -rf /important"}
    if myToolCallHandler(e).Permit {
        t.Fatal("expected dangerous command to be denied")
    }
}
```

## Architecture

```text
github.com/Checkmarx/ast-cx-hooks
├── agenthooks.go        # Core: AddRoute, Dispatch, Process/ProcessE, RouteNames
├── unified.go           # The 8 unified hooks → thin registry over platform adapters
├── registry.go          # Generic registerAdapters wiring
├── route_catalog.go     # Single source of truth: route → settings file / event key / style
├── scenarios.go         # Per-hook registries + named-scenario dispatch (fail-closed gates)
├── aliases.go           # Re-exports the hookcore vocabulary as the public API
├── install/             # Importable installer (InstallClaude/…, CmdForFunc) — Catalog-driven
├── internal/hookcore/   # Leaf vocabulary: events, verdicts, Run/RunE, ToolConvention (tool naming)
├── claude|…|copilotcli/ # Per-platform types, response builders, and adapters.go (translation)
├── internal/scaffold/   # Templates + generator for `agenthooks init`
└── cmd/agenthooks/      # CLI: init · build · install
```

**Flow:** the agent invokes `myhook <route>` → `Dispatch` looks up the route → the platform's
`adapters.go` decodes the event, builds a unified event, calls your handler, and maps the verdict
back to that platform's response — all via `hookcore.Run`/`RunE`. The unified vocabulary lives in
the leaf `internal/hookcore` package so every platform package shares it without an import cycle,
and the public API stays `agenthooks.*`.

## API reference

**Core:** `AddRoute(name, fn)` · `Dispatch()` · `Process(handler)` · `ProcessE(handler)` (returned error blocks via exit `2`) · `RouteNames()`

**Unified hooks:** `WhenAgentIdle` · `WhenSubagentIdle` · `BeforeToolCall` · `BeforeFileEdit` · `AfterFileWrite` · `AfterToolFailure` · `BeforeFileRead` · `BeforePrompt` (each with a `…Scenario` variant)

**Event helpers:** `AgentIdleEvent.IsLooping()` · `ToolCallEvent.IsShell()` · `ToolCallEvent.IsMCP()`

## Security model

These hooks gate agent actions, so failure behavior matters:

- **Malformed / unparseable stdin → fail OPEN.** A hook that cannot decode its input logs to
  stderr and exits 0 with no decision, so a bad payload never blocks the agent. If you rely on a
  hook as a hard control, monitor its stderr — a wire-format change could otherwise silently
  disable the gate.
- **Misconfigured scenarios → fail CLOSED.** A gating hook (`BeforeToolCall`, `BeforeFileEdit`,
  `BeforeFileRead`, `BeforePrompt`) with named scenarios but no default and no match **denies**
  rather than allowing, and logs why.
- **`install` is non-destructive.** It merges into existing settings (preserving your other
  hooks), aborts rather than overwriting a file it cannot parse, and writes a `.bak` first.

## Documentation

- [Usage Guide](docs/usage.md) — per-agent payloads, routes, and testing hooks locally.
- [Troubleshooting](docs/troubleshooting.md) — common build, install, and runtime issues.
- [Changelog](CHANGELOG.md) — release history.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, module architecture, and
contribution guidelines. Please also read our [Code of Conduct](docs/CODE_OF_CONDUCT.md).
Maintainers are listed in [MAINTAINERS.md](MAINTAINERS.md) and code ownership in
[CODEOWNERS.txt](CODEOWNERS.txt).

## Security

To report a vulnerability, follow the process in [SECURITY.md](SECURITY.md) — please do not
open a public issue for security reports. For the library's runtime failure behavior, see the
[Security model](#security-model) above.

## License

Apache 2.0 — see [LICENSE.txt](LICENSE.txt) for details.

---

Website: [Checkmarx](https://checkmarx.com).

© 2026 Checkmarx Ltd. All Rights Reserved.

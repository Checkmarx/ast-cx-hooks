<div align="center">

# 🪝 cxagenthooks

### One hook codebase. Every AI coding agent.

Write a single handler, compile one binary, and it runs across
**Claude Code · Cursor · Windsurf Cascade · Factory Droid · Gemini CLI · GitHub Copilot (VS Code) · GitHub Copilot CLI**.

[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Agents](https://img.shields.io/badge/agents-7-4F46E5)](#-supported-agents)
[![Unified hooks](https://img.shields.io/badge/unified%20hooks-7-7C3AED)](#-unified-hooks)
[![Dependencies](https://img.shields.io/badge/dependencies-zero-22C55E)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/CheckmarxDev/ast-cx-hooks.svg)](https://pkg.go.dev/github.com/CheckmarxDev/ast-cx-hooks)
[![Status](https://img.shields.io/badge/status-release%20candidate-F59E0B)](#-project-status)

</div>

```go
package main

import (
    "strings"

    "github.com/CheckmarxDev/ast-cx-hooks"
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

> That one handler now gates shell commands in **all seven** agents — no per-platform code.

---

## Contents

- [✨ Why cxagenthooks?](#-why-cxagenthooks)
- [📦 Installation](#-installation)
- [🤖 Supported agents](#-supported-agents)
- [🧩 Unified hooks](#-unified-hooks)
- [🗺️ Support matrix](#️-support-matrix)
- [👥 Multiple implementations per hook](#-multiple-implementations-per-hook-scenarios)
- [🛠️ CLI: scaffold, build, install](#️-cli-scaffold-build-install)
- [🧱 Platform-specific hooks](#-platform-specific-hooks)
- [🧪 Testing](#-testing)
- [🏗️ Architecture](#️-architecture)
- [📚 API reference](#-api-reference)
- [📖 More docs](#-more-docs)
- [🚦 Project status](#-project-status)

---

## ✨ Why cxagenthooks?

Every AI coding agent ships its own hook system — different JSON schemas, response
formats, blocking semantics, and config files. `cxagenthooks` collapses all of that
into **one** surface:

| | Without cxagenthooks | With cxagenthooks |
|---|---|---|
| **Handlers** | 7 separate implementations | 1 unified handler |
| **JSON schemas** | Learn 7 different formats | Learn 1 event struct |
| **Config files** | Hand-maintain 7 configs | `agenthooks install` writes them |
| **Binaries** | Build per platform manually | `agenthooks build` cross-compiles |
| **Dependencies** | — | **Zero** (stdlib only) |

---

## 📦 Installation

```bash
go get github.com/CheckmarxDev/ast-cx-hooks
```

---

## 🤖 Supported agents

| Agent | Config location | Output style |
|---|---|---|
| **Claude Code** | `~/.claude/settings.json` | nested JSON decision |
| **Cursor** | `~/.cursor/hooks.json` | flat JSON decision |
| **Windsurf Cascade** | `~/.codeium/windsurf/hooks.json` | exit code `2` only |
| **Factory Droid** | `~/.factory/settings.json` | nested JSON / exit `2` |
| **Gemini CLI** | `~/.gemini/settings.json` | JSON decision / exit `2` |
| **GitHub Copilot (VS Code)** _(Preview)_ | `.github/hooks/*.json` (project) | nested JSON decision |
| **GitHub Copilot CLI** | `~/.copilot/hooks/agenthooks.json` | **flat** JSON / exit `2` |

> The VS Code Copilot extension and the GitHub Copilot **CLI** are different products with
> incompatible hook schemas (nested vs. flat output, `updatedInput` vs. `modifiedArgs`,
> `runTerminalCommand` vs. `bash`), so they are separate packages — `copilot` and `copilotcli`.

---

## 🧩 Unified hooks

**7 unified hook categories** map automatically to each platform's native events.
Write the handler once; `cxagenthooks` translates the wire format per agent.

<details open>
<summary><b><code>WhenAgentIdle</code> — the agent finished responding</b></summary>

```go
agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
    if e.IsLooping() {
        return agenthooks.Resume() // break infinite continuation loops
    }
    return agenthooks.Interrupt("Please run the tests before finishing.")
})
```

**Verdicts:** `Resume()` · `Interrupt(feedback)`
</details>

<details>
<summary><b><code>BeforeToolCall</code> — gate tool & command execution</b></summary>

```go
agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    if e.IsShell() {
        return agenthooks.AskUser("Please confirm this shell command.")
    }
    return agenthooks.AllowWithInput(sanitize(e.ToolArgs)) // rewrite tool input before it runs
})
```

**Verdicts:** `Allow()` · `AllowWithNote(msg)` · `AllowWithInput(json)` · `AskUser(reason)` · `Deny(reason)`
</details>

<details>
<summary><b><code>AfterFileWrite</code> — react to file edits</b></summary>

```go
agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
    if strings.HasSuffix(e.FilePath, ".go") {
        return agenthooks.AnnotateWrite("Reminder: run `go vet` before committing.")
    }
    return agenthooks.AcceptWrite()
})
```

**Verdicts:** `AcceptWrite()` · `RejectWrite(reason)` · `AnnotateWrite(note)`
</details>

<details>
<summary><b><code>BeforePrompt</code> — filter or enrich user prompts</b></summary>

```go
agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
    if containsSecret(e.Text) {
        return agenthooks.RejectPrompt("Prompt contains sensitive information.")
    }
    return agenthooks.EnrichPrompt("Always follow our coding standards.")
})
```

**Verdicts:** `AcceptPrompt()` · `RejectPrompt(msg)` · `EnrichPrompt(context)`
</details>

<details>
<summary><b><code>WhenSubagentIdle</code> — gate subagent completion</b></summary>

```go
agenthooks.WhenSubagentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
    if e.IsLooping() {
        return agenthooks.Resume()
    }
    return agenthooks.Interrupt("Subagent must summarise its findings before stopping.")
})
```

**Verdicts:** `Resume()` · `Interrupt(feedback)` (reuses the idle verdict)
</details>

<details>
<summary><b><code>AfterToolFailure</code> — react to failed tool calls</b></summary>

```go
agenthooks.AfterToolFailure(func(e agenthooks.ToolFailureEvent) agenthooks.ToolFailureVerdict {
    return agenthooks.AnnotateFailure("Tool " + e.ToolName + " failed: " + e.Error)
})
```

**Verdicts:** `AcknowledgeFailure()` · `AnnotateFailure(note)` · `RejectAfterFailure(reason)`
</details>

<details>
<summary><b><code>BeforeFileRead</code> — gate the agent reading a file</b></summary>

```go
agenthooks.BeforeFileRead(func(e agenthooks.FileReadEvent) agenthooks.FileReadVerdict {
    if strings.HasSuffix(e.FilePath, ".env") {
        return agenthooks.DenyRead("Reading secret files is not allowed.")
    }
    return agenthooks.AllowRead()
})
```

**Verdicts:** `AllowRead()` · `DenyRead(reason)`
</details>

---

## 🗺️ Support matrix

Which unified hooks each agent supports today:

| Unified hook | Claude | Cursor | Windsurf | Droid | Gemini | Copilot | Copilot CLI |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `WhenAgentIdle` | ✅ | ✅ | ✅ ¹ | ✅ | ✅ | ✅ | ✅ |
| `BeforeToolCall` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `AfterFileWrite` | ✅ | ✅ ¹ | ✅ ¹ | ✅ | ✅ | ✅ | ✅ |
| `BeforePrompt` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ ³ |
| `WhenSubagentIdle` | ✅ | ✅ | — | ✅ | — | ✅ | ✅ |
| `AfterToolFailure` | ✅ | ✅ ² | — | — | — | — | ✅ |
| `BeforeFileRead` | — | ✅ | ✅ | — | — | — | — |

<sub>¹ fire-and-forget — feedback is logged, not enforced by the agent.  ² observational on Cursor — the verdict is ignored.  ³ observational on Copilot CLI — `userPromptSubmitted` output is not processed, so a reject is logged, not enforced.</sub>

---

## 👥 Multiple implementations per hook (scenarios)

Several teams — or scenarios — can share **one** hook. Register the **default** with
`BeforeToolCall(fn)` and any number of **named scenarios** with `BeforeToolCallScenario(name, fn)`.
The caller passes a **scenario key**; the framework runs the matching handler (or the default).

```go
func main() {
    // Team Phoenix and Team Cypher each own their own pre-tool-call logic:
    agenthooks.BeforeToolCallScenario("phoenix", phoenix.Handle)
    agenthooks.BeforeToolCallScenario("cypher",  cypher.Handle)

    // Optional default — runs when no scenario key is supplied or matched:
    agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
        return agenthooks.Allow()
    })

    agenthooks.Dispatch()
}
```

**How the caller supplies the key** (default selector = `ScenarioFromArg`):

| Channel | How |
|---|---|
| **CLI arg** | `myhook claude-pre-tool-use phoenix` |
| **Env var** | `AGENTHOOKS_SCENARIO=phoenix myhook claude-pre-tool-use` |
| **Event content** | `agenthooks.UseScenarioSelector(func(m agenthooks.ScenarioMeta) string { if strings.Contains(m.WorkDir, "phoenix") { return "phoenix" }; return "" })` |

Resolution order: a matching named scenario → the registered default → a permissive fallback verdict.
Every unified hook has a `…Scenario` variant — `WhenAgentIdleScenario`, `WhenSubagentIdleScenario`,
`BeforeToolCallScenario`, `AfterToolFailureScenario`, `AfterFileWriteScenario`, `BeforeFileReadScenario`,
`BeforePromptScenario`.

---

## 🛠️ CLI: scaffold, build, install

```bash
# 1. Scaffold a starter hooks project (main.go + README + .gitignore + policy.json)
go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks init --dir ./my-hooks

# 2. Build — current platform, or cross-compile to dist/ (macOS/Linux/Windows × amd64/arm64)
go build -o myhook .
go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks build

# 3. Install into every agent's settings file in one shot
go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks install ./myhook
```

`install` writes the correct config — in each agent's own shape — to:

`~/.claude/settings.json` · `~/.cursor/hooks.json` · `~/.codeium/windsurf/hooks.json` · `~/.factory/settings.json` · `~/.gemini/settings.json` · `~/.copilot/hooks/agenthooks.json`

> The **VS Code Copilot extension** is project-scoped (`.github/hooks/*.json`), so it's set up by
> hand — see [Testing → Copilot](#github-copilot-vs-code-preview). The **Copilot CLI** uses a
> home-directory hooks file, so `install` writes it automatically.

---

## 🧱 Platform-specific hooks

Need raw, per-platform event data? Use `AddRoute` with a platform package:

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

| Package | Import path |
|---|---|
| Claude Code | `…/ast-cx-hooks/claude` |
| Cursor | `…/ast-cx-hooks/cursor` |
| Windsurf | `…/ast-cx-hooks/windsurf` |
| Factory Droid | `…/ast-cx-hooks/droid` |
| Gemini CLI | `…/ast-cx-hooks/gemini` |
| VS Code Copilot _(Preview)_ | `…/ast-cx-hooks/copilot` |
| GitHub Copilot CLI | `…/ast-cx-hooks/copilotcli` |

Each package models that agent's full event surface and ships response builders
(`additionalContext`, tool-input/output rewrite, permission decisions, and more).

---

## 🧪 Testing

### Drive a hook by hand

Hook binaries read JSON from stdin and write JSON to stdout; the first arg is the **route**:

```bash
go build -o myhook .

# Claude pre-tool-use
echo '{"tool_name":"Bash","tool_input":{"command":"rm -rf /"},"session_id":"t","cwd":"/tmp"}' | ./myhook claude-pre-tool-use

# Cursor stop
echo '{"status":"completed","loop_count":0,"conversation_id":"t"}' | ./myhook cursor-stop

# Gemini before-tool
echo '{"tool_name":"run_shell_command","tool_input":{"command":"ls"},"session_id":"t"}' | ./myhook gemini-before-tool

# Copilot CLI pre-tool-use (note: lowercase tool names + FLAT output)
echo '{"hook_event_name":"PreToolUse","tool_name":"bash","tool_input":{"command":"rm -rf /"}}' | ./myhook copilot-cli-pre-tool-use
```

### Live agents

| Agent | Steps |
|---|---|
| **Claude / Cursor / Droid** | `install`, open the agent, trigger a hooked action; config lands in the agent's settings file. |
| **Windsurf** | `install`; pre-hooks block via **exit 2**, post-hooks are fire-and-forget. |
| **Gemini** | `install` auto-writes `~/.gemini/settings.json` (matcher-group shape), then run `gemini`. |
| **Copilot CLI** | `install` auto-writes `~/.copilot/hooks/agenthooks.json` (`{"version":1,"hooks":{…}}`), then run `copilot`. |

#### GitHub Copilot (VS Code) _(Preview)_

Copilot hooks are project-scoped, so add the config by hand — workspace `.github/hooks/copilot.json`:

```json
{
  "hooks": {
    "PreToolUse":       [{ "type": "command", "command": "/path/to/myhook copilot-pre-tool-use" }],
    "PostToolUse":      [{ "type": "command", "command": "/path/to/myhook copilot-after-file-write" }],
    "UserPromptSubmit": [{ "type": "command", "command": "/path/to/myhook copilot-user-prompt-submit" }],
    "Stop":             [{ "type": "command", "command": "/path/to/myhook copilot-stop" }]
  }
}
```

> VS Code Copilot also reads `~/.claude/settings.json`, so your Claude routes fire for Copilot too — but the Copilot routes above use the correct field shapes (Copilot uses camelCase `sessionId`) and isolate per-agent policy.

> **Copilot CLI is a separate product** from the VS Code extension — use the `copilot-cli-*` routes
> (auto-installed to `~/.copilot/hooks/agenthooks.json`), not the `copilot-*` ones. Its output is
> flat (`{"permissionDecision":…}`, no `hookSpecificOutput` wrapper) and its tools are lowercase
> (`bash`, `create`, `edit`).

### Unit-test your handler

```go
func TestDenyDangerousCommands(t *testing.T) {
    e := agenthooks.ToolCallEvent{Kind: agenthooks.ToolKindShell, Command: "rm -rf /important"}
    if myToolCallHandler(e).Permit {
        t.Fatal("expected dangerous command to be denied")
    }
}
```

---

## 🏗️ Architecture

```text
github.com/CheckmarxDev/ast-cx-hooks
├── agenthooks.go        # Core: AddRoute, Dispatch, Process/ProcessE, RouteNames
├── unified.go           # The 7 unified hooks → thin registry over platform adapters
├── registry.go          # Generic registerAdapters wiring
├── route_catalog.go     # Single source of truth: route → settings file / event key / style
├── scenarios.go         # Per-hook registries + named-scenario dispatch (fail-closed gates)
├── aliases.go           # Re-exports the hookcore vocabulary as the public API
├── internal/hookcore/   # Leaf vocabulary: events, verdicts, Run/RunE, ToolConvention (tool naming)
├── claude|…|copilotcli/ # Per-platform types, response builders, and adapters.go (translation)
├── internal/codec/      # JSON stdin/stdout serialization
├── internal/scaffold/   # Templates + generator for `agenthooks init`
└── cmd/agenthooks/      # CLI: init · build · install
```

**Flow:** the agent invokes `myhook <route>` → `Dispatch` looks up the route → the
platform's `adapters.go` decodes the event, builds a unified event, calls your handler,
and maps the verdict back to that platform's response — all via `hookcore.Run`/`RunE`.

The unified vocabulary lives in the leaf `internal/hookcore` package, so every platform
package can share it without an import cycle, and the public API stays `agenthooks.*`.

---

## 📚 API reference

**Core**

| Function | Description |
|---|---|
| `AddRoute(name, fn)` | Register a handler for a route name |
| `Dispatch()` | Run the handler matching `os.Args[1]` |
| `Process(handler)` | JSON stdin → handler → JSON stdout |
| `ProcessE(handler)` | Like `Process`, but a returned error blocks via exit `2` |
| `RouteNames()` | List all registered route names (inspection/tests) |

**Unified hooks**

| Function | Fires when |
|---|---|
| `WhenAgentIdle(fn)` | the agent finishes responding |
| `WhenSubagentIdle(fn)` | a subagent finishes |
| `BeforeToolCall(fn)` | before a tool/command executes |
| `AfterToolFailure(fn)` | a tool call fails |
| `AfterFileWrite(fn)` | after a file is written/edited |
| `BeforeFileRead(fn)` | before the agent reads a file |
| `BeforePrompt(fn)` | before a user prompt is processed |

**Event helpers**

| Method | On | Description |
|---|---|---|
| `e.IsLooping()` | `AgentIdleEvent` | continuation-loop detection |
| `e.IsShell()` | `ToolCallEvent` | is this a shell command? |
| `e.IsMCP()` | `ToolCallEvent` | is this an MCP tool call? |

---

## 📖 More docs

| Doc | Purpose |
|-----|---------|
| [docs/SCHEMA-REFERENCE.md](docs/SCHEMA-REFERENCE.md) | Field-by-field reference: every agent's input/output JSON tags, route map, tool-naming conventions — for diffing against the official docs |
| [docs/WHATS-NEW.md](docs/WHATS-NEW.md) | Architecture before/after + every extra hook, event, field, builder, and bug fix |
| [docs/CONFLUENCE-PAGE.md](docs/CONFLUENCE-PAGE.md) | Stakeholder narrative & route reference |
| [docs/DEMO.md](docs/DEMO.md) | Multi-agent demo: prep, scripted stdin tests, live steps |
| [docs/VIDEO-STORYBOARD.md](docs/VIDEO-STORYBOARD.md) | Short demo-video storyboard |

---

## 🔒 Security model

These hooks gate agent actions, so the failure behavior matters:

- **Malformed / unparseable stdin → fail OPEN.** If a hook cannot decode its input it logs
  to stderr and exits 0 with no decision, so a bad payload never *blocks* the agent
  (availability over enforcement). If you rely on a hook as a hard control, monitor its
  stderr and treat decode errors as alerts — a wire-format change could otherwise silently
  disable the gate.
- **Misconfigured scenarios → fail CLOSED.** If a gating hook (`BeforeToolCall`,
  `BeforeFileRead`, `BeforePrompt`) has named scenarios but no default and none match the
  key, the framework **denies** rather than allowing, and logs why.
- **`agenthooks install` is non-destructive.** It merges into existing settings (preserving
  your other hooks), aborts rather than overwriting a file it cannot parse, and writes a
  `.bak` before modifying.
- **Scaffold/example policies are illustrative.** The substring blocklists in `agenthooks init`
  output and `examples/` are demos and trivially bypassable — real policies should parse and
  normalize commands and prefer allowlists.

---

## 🚦 Project status

**Release candidate.** Zero dependencies; layered architecture, fully unit-tested (the adapter
seam is exercised end-to-end through `Dispatch`, including the exit-2 blocking path; the tool-naming
`ToolConvention` table has a dedicated unit test; install has golden + merge-safety tests). CI runs
`gofmt`/`build`/`vet`/`test -race` on every push. Every agent's input/output fields have been
audited field-by-field against the official live docs.

Before a production rollout: (1) **add a LICENSE** (none yet — required to be legally importable),
and (2) **smoke-test each agent live**. A small number of fields are modeled best-effort because the
vendor doc leaves them unspecified — notably the GitHub Copilot CLI `create`/`edit` tool input keys
(the doc types `tool_input` as `unknown`); these are captured as raw JSON and the diff extraction is
a documented best-effort guess. See [docs/SCHEMA-REFERENCE.md](docs/SCHEMA-REFERENCE.md) for the full
field reference and [docs/WHATS-NEW.md](docs/WHATS-NEW.md) for the verification checklist.
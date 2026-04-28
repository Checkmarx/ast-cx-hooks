# cxagenthooks

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
- `~/.gemini/settings.json`

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

    // React to file edits
    agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
        if strings.HasSuffix(e.FilePath, ".go") {
            return agenthooks.AnnotateWrite("Remember to run go vet.")
        }
        return agenthooks.AcceptWrite()
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
echo '{"tool_name":"execute_bash","tool_input":{"command":"ls"},"session_id":"test","cwd":"/tmp"}' | ./myhook gemini-before-tool
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

1. Build and install:
   ```bash
   go build -o myhook .
   go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks install ./myhook
   ```
2. Open Gemini CLI — hooks are configured in `~/.gemini/settings.json`.
3. Note: Gemini pre-hooks block via **exit code 2**.

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

## Checkmarx MCP Server

The `mcp/` package provides a local **Model Context Protocol (MCP) server** that exposes Checkmarx security guardrails as tools callable by any MCP-compatible agent.

### Starting the MCP server

```bash
cx mcp
```

The server runs over **stdio** and is compatible with Claude Desktop, Cursor, VS Code Copilot, and Windsurf.

### Configuration (Claude Desktop)

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "checkmarx": { "command": "cx", "args": ["mcp"] }
  }
}
```

### Available MCP tools

| Tool | Input | Output | Purpose |
|------|-------|--------|---------|
| `cx_shell_guard` | `{"command": "..."}` | `{"allowed": bool, "reason": "..."}` | Check a shell command against the blocklist policy |
| `cx_prompt_guard` | `{"text": "..."}` | `{"clean": bool, "blocked": bool, "reason": "..."}` | Scan a prompt for secrets, policy patterns, and restricted paths |

The server injects instructions into the AI's system prompt to ensure it calls the guard tools before executing commands or sending external data. Both tools **fail-open** — if the policy file is missing or the license check fails, all requests are allowed.

### Pattern syntax reference

Different policy fields use different matching rules. The table below is authoritative — pick your pattern syntax based on which field you're editing.

| Field | Match type | Wildcards supported | Case sensitivity |
|---|---|---|---|
| `restricted_files` | literal + basename + doublestar glob | `*`, `?`, `[...]`, `**` | case-insensitive |
| `restricted_directories` | literal + prefix + doublestar glob | `*`, `?`, `[...]`, `**` | case-insensitive |
| `allowed_files` | literal + basename + doublestar glob | `*`, `?`, `[...]`, `**` | case-insensitive |
| `allowed_directories` | literal + prefix + doublestar glob | `*`, `?`, `[...]`, `**` | case-insensitive |
| `args_include` | literal + single-segment glob (`path.Match`) | `*`, `?`, `[...]` only — **no `**`** | case-insensitive |
| `args_exclude` | **substring containment** (`strings.Contains`) | **none — `*` is literal** | case-insensitive |

All path fields are normalized to lowercase and forward-slashes before matching, so Windows policies can use `\\` or `/` interchangeably.

#### File list fields — `restricted_files`, `allowed_files`

A pattern matches a path when ANY of the following is true:

- Lowercased forward-slash forms are equal (literal path: `/etc/passwd`, `C:\Windows\System32\drivers\etc\hosts`)
- The pattern equals the basename of the path (bare name: `kubeconfig`, `terraform.tfstate`, `.env`)
- The path has suffix `/` + pattern (back-compat for paths with directory segments)
- The pattern contains `*`, `?`, or `[...]` and doublestar matches the full path (`**/*.pem` matches `/srv/keys/cert.pem`)
- The pattern contains `*`, `?`, or `[...]` and doublestar matches the basename (`*.pem` matches `foo.pem`)

**Supported:**

| Pattern | Matches | Rationale |
|---|---|---|
| `.env` | `/app/.env`, `/home/a/.env`, `.env` | basename equality |
| `kubeconfig` | `~/.kube/kubeconfig` | basename equality |
| `*.pem` | `foo.pem`, `key.pem` | basename glob |
| `.env*` | `.env`, `.env.local`, `.env.production` | basename glob |
| `id_rsa*` | `id_rsa`, `id_rsa.pub` | basename glob |
| `**/*.pem` | `/any/depth/cert.pem`, `certs/foo.pem` | doublestar full-path |
| `**/.env` | `/svc/api/.env` | doublestar full-path |
| `/home/*/projects/**/*.java` | `/home/alice/projects/app/src/Foo.java` | doublestar full-path |

**Not supported (silently mis-matches — don't use):**

| Pattern | Why it won't do what you expect |
|---|---|
| `~/.ssh/id_rsa` | `~` is NOT expanded to the user's home; treated as the literal character `~` |
| `%USERPROFILE%\.ssh\id_rsa` | Environment variables are NOT expanded; treated as literal `%USERPROFILE%` |
| `{a,b}.pem` | Brace expansion is NOT supported by our matcher; treated as literal `{a,b}.pem` |

#### Directory list fields — `restricted_directories`, `allowed_directories`

A pattern matches a path when ANY of the following is true:

- Lowercased forward-slash forms are equal (path IS the directory)
- Path starts with `pattern + "/"` (literal prefix match — path is nested inside the directory)
- Pattern contains glob metacharacters and doublestar matches the path directly
- Pattern contains glob metacharacters and doublestar matches `pattern + "/**"` (path is anywhere under the glob-matched directory)

**Supported:**

| Pattern | Matches | Rationale |
|---|---|---|
| `/etc/` | `/etc`, `/etc/passwd`, `/etc/ssh/sshd_config` | literal prefix |
| `C:\Windows\System32\` | `C:/Windows/System32`, `C:/Windows/System32/drivers/etc/hosts` | literal prefix, normalised |
| `/home/*/.ssh` | `/home/alice/.ssh`, `/home/alice/.ssh/id_rsa` | single-segment glob + `/**` extension |
| `C:/Users/*/.ssh` | `C:/Users/alice/.ssh/id_rsa` | single-segment glob |
| `**/secrets/**` | `/repo/secrets/db.yaml`, `/a/b/secrets/c/d.txt` | doublestar on both sides |
| `**/node_modules` | anywhere a `node_modules` dir appears | doublestar prefix |

**Not supported:**

| Pattern | Why it won't work |
|---|---|
| `~/.ssh`, `~/Projects` | `~` not expanded |
| `%USERPROFILE%\Projects` | `%USERPROFILE%` not expanded |
| Trailing slash differences | Leading/trailing slashes are normalised; `/etc` and `/etc/` behave identically |

#### Argument whitelist — `args_include`

Uses Go's stdlib `path.Match` per token, case-insensitively. Each token of the command (skipping the command name) must match at least one pattern. Unmatched tokens trigger an **ask**, not a hard block.

**Supported:**

| Pattern | Matches | Misses |
|---|---|---|
| `compile` | exactly `compile` | `compile-goal` |
| `-D*` | `-Dfoo`, `-Dmaven.compiler.source` | `--Dfoo` (single-segment only) |
| `--*` | `--offline`, `--batch-mode` | `-o` |
| `-P?` | `-PA`, `-PB` | `-Pfoo` (`?` is ONE char only) |
| `[abc]*` | tokens starting with `a`, `b`, or `c` | uppercase tokens |

**Not supported:**

- `**` — has no meaningful meaning for tokens (which don't contain `/`); behaves the same as `*`
- Brace expansion `{compile,test}` — use two separate entries instead
- Substring matching — the pattern must match the WHOLE token (wrap with `*…*` if you want that)

#### Argument blacklist — `args_exclude`

**Does NOT use glob syntax.** Each entry is matched as a case-insensitive substring of the ENTIRE command line (not per-token). A pattern containing `*` is treated as a literal string.

Why it's a substring denylist rather than a glob:

- A denylist should be **broader** than tokenised matching, not narrower. `"deploy"` must catch `mvn deploy`, `mvn deploy:deploy-file`, and `mvn my-deploy-wrapper` — all of which a per-token glob would miss.
- The full-command sweep means patterns like `-Durl=`, `-Dgpg.passphrase`, `-DaltDeploymentRepository` catch the argument regardless of whether it's written as `-Durl=http://…` or `-Durl http://…`.

**Supported:**

| Pattern | Blocks |
|---|---|
| `deploy` | `mvn deploy`, `mvn deploy:deploy-file`, `mvn my-custom-deployment` |
| `-DskipTests` | `mvn compile -DskipTests`, `mvn -DskipTests=true compile` |
| `-Durl=` | `-Durl=http://evil.com` |
| `exec:java` | `mvn exec:java` |
| `maven-antrun-plugin` | any mvn invocation referencing the plugin |

**Not supported (treated as literal characters):**

| Pattern | What actually happens |
|---|---|
| `-Dmaven.test.skip*` | Looks for the literal string `-Dmaven.test.skip*` in the command — effectively never matches |
| `deploy*` | Looks for `deploy*` verbatim — misses plain `deploy` |
| `*deploy*` | Misses `deploy` (no leading/trailing `*` in the command) |

If you need a narrower match than substring (e.g. block `deploy` but NOT `my-deployment`), either:
- Anchor the substring with spaces: `" deploy "` will only match when `deploy` is surrounded by whitespace, OR
- Use a more specific substring such as `mvn deploy` or `deploy:deploy-file` or `deploy ` (note the trailing space).

### Common pitfalls

- **`~` and `%USERPROFILE%` are never expanded** anywhere in the policy. Use explicit paths, doublestar (`/home/*/.ssh`), or OS-specific sections (`linux` / `mac` / `windows` arrays).
- **`*` alone has no meaning for `args_exclude`.** It's a literal `*` character.
- **Basename match is ONLY for `_files` fields, not `_directories`.** `restricted_directories: ["secrets"]` will NOT block `/a/b/secrets/c.yaml` — use `"**/secrets/**"` or a full prefix.
- **Tool-rule `allowed_files` only checks file-like tokens** (those containing `./\`) so plain argument words like `compile` or `test` are not validated against the list. They go through `args_include` instead.

### Policy file

Guards read policy from `~/.checkmarx/policyhooks.json`. See [guardrails/policy.go](guardrails/policy.go) for the full schema. Example:

```json
{
  "default_policy": {
    "blocklist_tools": {
      "enabled": true,
      "tools": [
        { "name": "curl", "os": ["linux", "darwin"], "category": "network", "risk": "data-exfiltration" }
      ]
    },
    "context_policy": {
      "enabled": true,
      "content_scanning": {
        "enabled": true,
        "patterns": [
          { "id": "no-prod-urls", "pattern": "https://prod\\.example\\.com", "description": "Production URLs" }
        ]
      }
    }
  }
}
```

## Architecture

```
github.com/CheckmarxDev/ast-cx-hooks
├── agenthooks.go          # Public API: re-exports from lifecycle/ and internal/dispatch/
├── lifecycle/             # Unified agent lifecycle hook handlers (all 5 agents)
│   ├── constants.go       #   AgentID and ToolKind constants
│   ├── idle.go            #   WhenAgentIdle — agent finished responding
│   ├── tool.go            #   BeforeToolCall — before tool/command executes
│   ├── filewrite.go       #   AfterFileWrite — after file is written/edited
│   └── prompt.go          #   BeforePrompt — before user prompt is processed
├── claude/                # Claude Code event types & response helpers
├── cursor/                # Cursor IDE event types & response helpers
├── windsurf/              # Windsurf Cascade event types & response helpers
├── droid/                 # Factory Droid event types & response helpers
├── gemini/                # Gemini CLI event types & response helpers
├── guardrails/            # Security policy engine (shell blocklist, prompt scanning)
├── cx/                    # Checkmarx guardrail registration & per-agent installers
├── mcp/                   # Local MCP server exposing guardrails as tools
├── internal/dispatch/     # Core routing: AddRoute, Dispatch, Process, ProcessE
├── internal/codec/        # JSON stdin/stdout serialization
├── internal/scaffold/     # Templates and generator for `agenthooks init`
└── cmd/agenthooks/        # CLI tool: init, install, and build commands
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

# Troubleshooting

Common problems when building, installing, and running **cxagenthooks**, with the exact
symptom and fix for each. If you are new to the library, start with the
[Usage Guide](usage.md).

## Table of Contents

- [Diagnosing quickly](#diagnosing-quickly)
- [Build & module issues](#build--module-issues)
- [Install issues](#install-issues)
- [Hook doesn't run at all](#hook-doesnt-run-at-all)
- [Hook runs but doesn't block](#hook-runs-but-doesnt-block)
- [`additionalContext` / feedback not showing up](#additionalcontext--feedback-not-showing-up)
- [Scenario handler not selected](#scenario-handler-not-selected)
- [Payloads and JSON errors](#payloads-and-json-errors)
- [Exit codes reference](#exit-codes-reference)

---

## Diagnosing quickly

Hook binaries read JSON from **stdin**, take the **route** as the first argument, and write
their response to **stdout** — errors and breadcrumbs go to **stderr**. To see what a hook
actually does, run it by hand and watch stderr:

```bash
echo '{"tool_name":"Bash","tool_input":{"command":"rm -rf /"},"session_id":"t"}' \
  | ./myhook claude-pre-tool-use
echo "exit: $?"
```

- **stdout** — the decision the agent will act on.
- **stderr** — decode errors, dropped/ignored verdicts, "no handler registered" messages.
- **exit code** — `0` = normal (decision is on stdout), `2` = block via error, `1` = no
  matching route. See [Exit codes reference](#exit-codes-reference).

> Because the library logs the important events to stderr, **watching stderr is the single
> most useful debugging step.** If you rely on a hook as a hard control, monitor its stderr
> in production — a wire-format change on the agent side could otherwise silently disable a
> gate (it fails **open** on unparseable input; see below).

---

## Build & module issues

**`go get` fails / wrong module path.** The import path is
`github.com/CheckmarxDev/ast-cx-hooks`. Make sure your project has a module first:

```bash
go mod init github.com/your-org/my-hooks
go get github.com/CheckmarxDev/ast-cx-hooks@latest
go mod tidy
```

**`go` version too old.** The library targets **Go 1.23+**. Check with `go version`; the
unified hooks use generics-based helpers that require a recent toolchain.

**`agenthooks build` fails for a target.** The `build` command shells out to `go build`
with `GOOS`/`GOARCH`/`CGO_ENABLED=0` for six targets (macOS, Linux, Windows × amd64/arm64).
It needs the `go` toolchain on `PATH`. Since the library is pure Go with zero third-party
dependencies, cross-compilation should succeed without a C toolchain — if a target fails,
the error names the exact `GOOS/GOARCH` pair.

**`agenthooks init` says a file already exists.** `init` refuses to overwrite. It aborts the
moment it finds an existing `main.go`, `README.md`, `.gitignore`, or `policy.json`. Point it
at an empty directory: `agenthooks init --dir ./fresh-dir`.

---

## Install issues

The installer prints one line per settings file it writes, e.g.
`✓ Claude Code (3 hooks) → .claude/settings.json`. Warnings are printed as
`warning: <path>: <reason>` and do **not** abort the other files.

**`warning: … refusing to overwrite …: existing file is not valid JSON`.** `install` is
non-destructive: if a target settings file already exists but isn't valid JSON, it **aborts
that file** rather than clobbering it. Fix the JSON (or move the file aside) and re-run.

**Where did my old config go?** Before modifying any non-empty settings file, `install`
writes a `<path>.bak` backup. Nested-style entries (Claude, Gemini, Copilot CLI) are
**appended and de-duplicated**, so your own hook entries for the same event are preserved.
The flat style (Cursor, Windsurf) is single-command-per-key by design.

**VS Code Copilot hooks weren't installed.** This is expected. `install` prints
`note: VS Code Copilot hooks are project-scoped (.github/hooks) — see README for manual setup`.
The VS Code Copilot extension reads hooks from `.github/hooks/*.json` in the project, not
from a home-directory settings file, so you wire it up by hand.

**The binary path in the config is wrong after moving it.** `install <binary>` records an
**absolute** path to the binary you pass. If you move or rename the binary, re-run
`install` so the settings point at the new location.

---

## Hook doesn't run at all

**`agenthooks: no handler registered for "<route>"`.** `Dispatch()` matched the route
argument against your registered handlers and found nothing. It then lists the available
routes on stderr and exits `1`. Causes:

- You registered a different unified hook than the route the agent invoked. A unified hook
  registers *all* the routes it maps to across platforms — e.g. `BeforeToolCall` covers
  `claude-pre-tool-use`, `cursor-before-shell`, `cursor-before-mcp`, and so on. Register the
  hook whose category matches the event.
- You used `AddRoute` for a platform route but with a typo in the name. Compare against the
  list printed on stderr.
- The agent's settings still point at an old binary. Re-run `agenthooks install ./myhook`.

**No first argument.** If invoked with no route argument, `Dispatch()` falls back to the
**binary's base name** as the route. That is almost never what you want — always invoke as
`myhook <route>`. This mostly bites when testing manually.

**The agent never invokes the hook.** Confirm the settings file for that agent actually
contains your command and that the event key matches. Run `install` (which writes the
correct per-agent shape) rather than hand-editing, and check the file it reported writing.

---

## Hook runs but doesn't block

Blocking semantics differ per agent — a verdict that blocks on Claude may only be logged on
another agent. This is by design; see the
[support matrix](../README.md#support-matrix) for exact coverage. Key cases:

- **Windsurf post-hooks are fire-and-forget.** `WhenAgentIdle` and `AfterFileWrite` on
  Windsurf log feedback but cannot enforce it. On stderr you'll see
  `agenthooks: windsurf <hook> is fire-and-forget; <detail> ignored`.
- **Cursor `AfterToolFailure` is observational** — the verdict is ignored.
- **Copilot CLI `BeforePrompt` is observational** — `userPromptSubmitted` output is not
  processed by the CLI.
- **Blocking pre-gates depend on the agent's channel.** `BeforeToolCall` / `BeforeFileEdit`
  block via a JSON decision on some agents and via **exit code `2`** on others (Windsurf,
  Droid, Gemini, Copilot CLI). If you drive the binary yourself and expect a block, check
  the exit code, not just stdout.

If a verdict is being dropped, the `agenthooks: … is fire-and-forget; … ignored` breadcrumb
on stderr tells you exactly which adapter dropped it and why.

---

## `additionalContext` / feedback not showing up

The extra "context" you attach with the `…WithContext` verdicts
(`DenyWithContext`, `RejectEditWithContext`, etc.) is only delivered where the agent's hook
protocol defines that field — **Claude and VS Code Copilot on the pre-tool / pre-file
gates**. On every other agent, a deny carries its **reason only**; the context is dropped
and logged. If your steering context isn't reaching the agent, confirm you're on a platform
that exposes the channel for that hook.

---

## Scenario handler not selected

If you registered named scenarios with the `…Scenario` variants but the wrong handler runs:

- **How the key is chosen (default selector `ScenarioFromArg`):** the **third CLI argument**
  wins (`myhook <route> <scenario>`), otherwise the `AGENTHOOKS_SCENARIO` environment
  variable. So `myhook claude-pre-tool-use phoenix` selects `phoenix`.
- **Resolution order:** matching named scenario → registered default → fallback verdict.
- **Gating hooks fail CLOSED.** For `BeforeToolCall`, `BeforeFileEdit`, `BeforeFileRead`,
  and `BeforePrompt`, a configuration with named scenarios but **no default and no match
  denies** the action rather than allowing it — and logs why. If actions are unexpectedly
  denied, check that the scenario key is actually being passed, or register a default.
- **Content-based selection:** to pick a scenario from the event body instead of an arg/env,
  set a selector with `UseScenarioSelector(func(m agenthooks.ScenarioMeta) string { … })`
  (`ScenarioMeta` carries `Agent`, `WorkDir`, and the raw event).

---

## Payloads and JSON errors

**`agenthooks: stdin decode error: …` and the hook exits 0.** The input on stdin wasn't
valid JSON for that route's event type, so the hook **failed open** — it logged to stderr
and exited `0` with no decision, so a malformed payload never blocks the agent. Common when
testing by hand:

- The JSON shape doesn't match the route. Each agent has its own field names. For example
  Copilot CLI uses **lowercase** tool names and a flat output shape:
  ```bash
  echo '{"hook_event_name":"PreToolUse","tool_name":"bash","tool_input":{"command":"rm -rf /"}}' \
    | ./myhook copilot-cli-pre-tool-use
  ```
- Shell quoting mangled the JSON. On Windows PowerShell, single-quote rules differ from
  bash — prefer a here-string or a file piped in with `Get-Content file.json | ./myhook <route>`.

**`agenthooks: stdout encode error: …` (exit 0).** Rare — the response couldn't be
marshaled. Usually indicates a malformed custom value passed into a verdict (e.g.
`AllowWithInput` with non-JSON-encodable content).

---

## Exit codes reference

| Exit code | Meaning |
|:---:|---|
| `0` | Normal run — the decision (if any) is on stdout. Also used on stdin/stdout **codec errors** (fail-open). |
| `1` | No handler registered for the route (`Dispatch` couldn't match), or a CLI command (`init`/`install`/`build`) failed. |
| `2` | **Blocking** signal — a `ProcessE` handler returned an error, or a unified verdict maps to exit-`2` blocking on that agent. The message is on stderr. |

---

Still stuck? Check the [Usage Guide](usage.md) for the full API, the
[README](../README.md) for the architecture and support matrix, and
[CONTRIBUTING.md](../CONTRIBUTING.md) for development setup.

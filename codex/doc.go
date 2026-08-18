// Package codex provides types and response helpers for OpenAI Codex CLI hooks.
//
// UNVERIFIED PACKAGE. Everything here is modeled from the published reference at
// https://learn.chatgpt.com/docs/hooks, not from a captured live hook payload.
// The doc presents Codex's hook JSON schema as a near-clone of Claude Code's
// (same field names: session_id, hook_event_name, cwd, continue, stopReason,
// permissionDecision, hookSpecificOutput.additionalContext, exit-code-2
// blocking), plus two extra common input fields (model, turn_id). That
// similarity is the basis for this package's shape, but the following specific
// points are inferred, not confirmed, and should be verified against a live
// payload before being relied on as a hard control:
//
//   - The shell tool name is assumed to be "Bash" (Claude's casing) — the doc
//     never gives shell/apply_patch/MCP tool names beyond examples in matcher
//     patterns.
//   - The MCP tool-name prefix is assumed to be "mcp__" (Claude's prefix) — the
//     doc does not state one.
//   - apply_patch's tool_input shape is assumed to be {"input": "<patch text>"},
//     inferred from public Codex apply_patch examples elsewhere, not from this
//     doc. Codex's apply_patch is a multi-file unified-patch tool, unlike
//     Claude's single-file Write/Edit, so there is no reliable per-file path or
//     diff — see internal/hookcore's CodexTools/codexDiff for the best-effort
//     handling (raw patch text surfaced as FileDiff.After, empty FilePath).
//   - PreToolUse output is documented with only "allow"/"deny" decision values
//     (no "ask", unlike Claude's allow/deny/ask/defer) — this package's
//     preToolDecision therefore never emits "ask" for Codex; see adapters.go.
//   - Stop/SubagentStop are not documented as carrying a stop_hook_active-style
//     loop-detection field, so none is modeled here.
//   - SubagentStart/SubagentStop field names beyond the event name itself are
//     assumed to mirror Claude's (agent_id, agent_type, ...).
//
// Event tiers, mirroring the two-tier model in claude/doc.go:
//
//   - Routed core — Stop, PreToolUse, PostToolUse, UserPromptSubmit, and
//     SubagentStop. These drive the unified agenthooks hooks, are written by
//     `agenthooks install`, and are exercised end-to-end through Dispatch.
//     PreToolUse backs both the generic tool-call gate and the pre-file-write
//     gate (scoped to apply_patch); PostToolUse backs the post-file-write hook
//     (also scoped to apply_patch). Codex documents no PostToolUseFailure event
//     and no file-read event, so AfterToolFailure and BeforeFileRead have no
//     Codex adapter — the same kind of gap Gemini/Windsurf/Droid already have
//     for one or more unified hooks.
//   - Extras — Codex-specific events with no unified-hook equivalent
//     (SessionStart, SessionEnd, SubagentStart, PermissionRequest, PreCompact,
//     PostCompact). They are provided for hand-coded routes via AddRoute +
//     Process. Their field tags track the official doc but, lacking
//     captured-payload tests, should be treated as best-effort until verified
//     against a live hook event.
//
// See https://learn.chatgpt.com/docs/hooks for the official reference.
package codex

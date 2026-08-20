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
//   - apply_patch's tool_input shape is CONFIRMED against a live payload to be
//     {"command": "<patch text>"} — the same key PreToolUse uses for shell
//     commands, not {"input": "<patch text>"} as originally assumed from public
//     examples ("input" is kept as a fallback). Codex's apply_patch is a
//     multi-file V4A-style patch tool, unlike Claude's single-file Write/Edit;
//     internal/hookcore's CodexTools scopes both the diff and the file path to
//     the FIRST file section in the patch ("*** Add/Update/Delete File: <path>"
//     — codexPatchFilePath), since FileDiff has no per-file grouping and mixing
//     hunks from multiple files would apply one file's edits against another
//     file's on-disk content. Within that first section, codexDiff
//     (internal/hookcore/conventions.go) reconstructs real before/after text
//     instead of surfacing the raw patch syntax: Add File strips each line's
//     leading "+" to rebuild the full new file; Update File emits one FileDiff
//     per "@@"-delimited hunk with context/"-"/"+" lines resolved into
//     Before/After, mirroring how Claude's Edit (old_string/new_string) is
//     applied. This is required for ASCA/KICS to scan actual source rather than
//     diff markers — confirmed against live multi-hunk Update File payloads
//     (see internal/hookcore/conventions_test.go). The full patch grammar
//     (Add/Delete/Update File, Move to, End of File, Environment ID) is
//     additionally cross-checked against openai/codex's own Lark grammar in
//     codex-rs/apply-patch/src/parser.rs — Delete File and Move to (rename)
//     have not been seen in a live Codex CLI payload yet, but their marker
//     text and grammar position are taken from that source, not guessed.
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

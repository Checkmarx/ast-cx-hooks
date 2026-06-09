// Package claude provides types and response helpers for Claude Code hooks.
//
// Claude Code hooks are defined in settings files (~/.claude/settings.json or
// .claude/settings.json) and execute shell commands at lifecycle points in the
// agent loop. Each hook process receives a JSON payload on stdin and writes a
// JSON decision to stdout.
//
// Event tiers. Two tiers of event types are modeled:
//
//   - Routed core — Stop, PreToolUse, PostToolUse, UserPromptSubmit, SubagentStop,
//     and PostToolUseFailure. These drive the unified agenthooks hooks, are written
//     by `agenthooks install`, and are exercised end-to-end through Dispatch.
//   - Extras — Claude-specific events with no unified-hook equivalent (Setup,
//     ConfigChange, CwdChanged, FileChanged, StopFailure, WorktreeCreate/Remove,
//     Elicitation, ElicitationResult, InstructionsLoaded, and so on). They are
//     provided for hand-coded routes via AddRoute + Process. Their field tags track
//     the official doc but, lacking captured-payload tests, should be treated as
//     best-effort until verified against a live hook event.
//
// See https://code.claude.com/docs/en/hooks for the official reference.
package claude

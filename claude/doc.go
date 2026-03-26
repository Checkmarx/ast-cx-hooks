// Package claude provides types and response helpers for Claude Code hooks.
//
// Claude Code hooks are defined in settings files (~/.claude/settings.json or
// .claude/settings.json) and execute shell commands at lifecycle points in the
// agent loop. Each hook process receives a JSON payload on stdin and writes a
// JSON decision to stdout.
//
// See https://code.claude.com/docs/en/hooks for the official reference.
package claude

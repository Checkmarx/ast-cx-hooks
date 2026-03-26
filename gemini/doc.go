// Package gemini provides types and response helpers for Gemini CLI hooks.
//
// Gemini CLI hooks are configured in settings.json (.gemini/settings.json or
// ~/.gemini/settings.json) and fire at lifecycle points in the agent loop.
// Hooks receive a JSON payload on stdin and communicate results via stdout/exit codes.
//
// See https://github.com/google-gemini/gemini-cli/blob/main/docs/hooks/reference.md
// for the official reference.
package gemini

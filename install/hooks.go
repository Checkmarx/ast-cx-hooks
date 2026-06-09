package install

// ClaudeHook builds a Claude/Droid-style hook group entry:
//
//	{ "hooks": [ { "type": "command", "command": cmd } ] }
//
// Claude Code and Factory Droid wrap each hook in a group object (with an
// optional matcher); the actual hook definition lives in the inner "hooks"
// array. This matches the documented Claude Code hook schema.
func ClaudeHook(cmd string) map[string]any {
	return map[string]any{
		"hooks": []any{
			map[string]any{"type": "command", "command": cmd},
		},
	}
}

// ClaudeHookWithMatcher is ClaudeHook plus a matcher regex applied to the
// tool name (e.g. "Write|Edit|MultiEdit"). Matchers scope the hook to a
// subset of tools — for example, fire a file-write guardrail only when the
// tool name matches a writing tool.
func ClaudeHookWithMatcher(cmd, matcher string) map[string]any {
	return map[string]any{
		"matcher": matcher,
		"hooks": []any{
			map[string]any{"type": "command", "command": cmd},
		},
	}
}

// CursorHook builds a Cursor-style hook entry: { "command": cmd }
func CursorHook(cmd string) map[string]any {
	return map[string]any{"command": cmd}
}

// WindsurfHook builds a Windsurf-style hook entry: { "command": cmd }.
// Structurally identical to CursorHook today; kept distinct so either
// vendor can diverge its schema without affecting the other.
func WindsurfHook(cmd string) map[string]any {
	return map[string]any{"command": cmd}
}

// GeminiHook builds a Gemini CLI-style hook group entry — same shape as
// ClaudeHook.
func GeminiHook(cmd string) map[string]any {
	return map[string]any{
		"hooks": []any{
			map[string]any{"type": "command", "command": cmd},
		},
	}
}

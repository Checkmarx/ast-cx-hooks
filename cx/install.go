package cx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Installers maps AgentID strings to their install functions.
// Callers can select any subset of agents to install for at runtime.
//
//	cx.Installers["cursor"](home, cxPath)           // install for one agent
//	for _, fn := range cx.Installers { fn(home, cx) } // install for all
var Installers = map[string]func(home, cx string) error{
	"claude":   InstallClaude,
	"cursor":   InstallCursor,
	"windsurf": InstallWindsurf,
	"droid":    InstallDroid,
	"gemini":   InstallGemini,
}

// InstallClaude patches ~/.claude/settings.json so Claude Code invokes
// "cx hooks <route>" on hook events.
func InstallClaude(home, cx string) error {
	return patchJSON(filepath.Join(home, ".claude", "settings.json"), func(m map[string]any) {
		hooks := ensureMap(m, "hooks")
		hooks["Stop"]             = []any{claudeHook(cx, "claude-stop")}
		hooks["PreToolUse"]       = []any{claudeHook(cx, "claude-pre-tool-use")}
		hooks["PostToolUse"]      = []any{claudeHook(cx, "claude-after-file-write")}
		hooks["UserPromptSubmit"] = []any{claudeHook(cx, "claude-user-prompt-submit")}
	})
}

// InstallCursor patches ~/.cursor/hooks.json so Cursor invokes
// "cx hooks <route>" on hook events.
func InstallCursor(home, cx string) error {
	return patchJSON(filepath.Join(home, ".cursor", "hooks.json"), func(m map[string]any) {
		m["version"] = 1

		hooks := ensureMap(m, "hooks")
		hooks["beforeSubmitPrompt"]   = []any{cursorHook(cx, "cursor-before-submit-prompt")}
		hooks["beforeShellExecution"] = []any{cursorHook(cx, "cursor-before-shell")}
		hooks["beforeMCPExecution"]   = []any{cursorHook(cx, "cursor-before-mcp")}
		hooks["afterFileEdit"]        = []any{cursorHook(cx, "cursor-after-file-edit")}
		hooks["stop"]                 = []any{cursorHook(cx, "cursor-stop")}
	})
}

// InstallWindsurf patches ~/.codeium/windsurf/hooks.json so Windsurf invokes
// "cx hooks <route>" on hook events.
func InstallWindsurf(home, cx string) error {
	return patchJSON(filepath.Join(home, ".codeium", "windsurf", "hooks.json"), func(m map[string]any) {
		hooks := ensureMap(m, "hooks")
		hooks["pre_run_command"]       = []any{windsurfHook(cx, "windsurf-pre-run-command")}
		hooks["pre_mcp_tool_use"]      = []any{windsurfHook(cx, "windsurf-pre-mcp-tool-use")}
		hooks["pre_user_prompt"]       = []any{windsurfHook(cx, "windsurf-pre-user-prompt")}
		hooks["post_write_code"]       = []any{windsurfHook(cx, "windsurf-post-write-code")}
		hooks["post_cascade_response"] = []any{windsurfHook(cx, "windsurf-post-cascade-response")}
	})
}

// InstallDroid patches ~/.factory/settings.json so Factory Droid invokes
// "cx hooks <route>" on hook events.
func InstallDroid(home, cx string) error {
	return patchJSON(filepath.Join(home, ".factory", "settings.json"), func(m map[string]any) {
		hooks := ensureMap(m, "hooks")
		hooks["Stop"]             = []any{claudeHook(cx, "droid-stop")}
		hooks["PreToolUse"]       = []any{claudeHook(cx, "droid-pre-tool-use")}
		hooks["PostToolUse"]      = []any{claudeHook(cx, "droid-after-file-write")}
		hooks["UserPromptSubmit"] = []any{claudeHook(cx, "droid-user-prompt-submit")}
	})
}

// InstallGemini patches ~/.gemini/settings.json so Gemini CLI invokes
// "cx hooks <route>" on hook events.
func InstallGemini(home, cx string) error {
	return patchJSON(filepath.Join(home, ".gemini", "settings.json"), func(m map[string]any) {
		hooks := ensureMap(m, "hooks")
		hooks["BeforeAgent"] = []any{geminiHook(cx, "gemini-before-agent")}
		hooks["BeforeTool"]  = []any{geminiHook(cx, "gemini-before-tool")}
		hooks["AfterTool"]   = []any{geminiHook(cx, "gemini-after-file-tool")}
		hooks["AfterAgent"]  = []any{geminiHook(cx, "gemini-after-agent")}
	})
}

// cmdString builds "cx hooks <route>", quoting the path if it has spaces.
// Backslashes are converted to forward slashes so the path survives bash -c
// evaluation on Windows (Git Bash / MSYS2).
func cmdString(cx, route string) string {
	cx = strings.ReplaceAll(cx, `\`, `/`)
	if strings.Contains(cx, " ") {
		cx = `"` + cx + `"`
	}
	return cx + " hooks " + route
}

// claudeHook builds a Claude/Droid-style hook group entry.
// Claude Code and Factory Droid require a two-level structure:
//
//	{ "hooks": [ { "type": "command", "command": "..." } ] }
//
// where the outer object is a hook group (with optional matcher) and
// the inner array contains the actual hook definitions.
func claudeHook(cx, route string) map[string]any {
	return map[string]any{
		"hooks": []any{
			map[string]any{"type": "command", "command": cmdString(cx, route)},
		},
	}
}

// cursorHook builds a Cursor-style hook entry: {command: "..."}
func cursorHook(cx, route string) map[string]any {
	return map[string]any{"command": cmdString(cx, route)}
}

// windsurfHook builds a Windsurf-style hook entry: {command: "..."}
func windsurfHook(cx, route string) map[string]any {
	return map[string]any{"command": cmdString(cx, route)}
}

// geminiHook builds a Gemini-style hook group entry.
// Gemini CLI requires the same two-level structure as Claude Code:
//
//	{ "hooks": [ { "type": "command", "command": "..." } ] }
func geminiHook(cx, route string) map[string]any {
	return map[string]any{
		"hooks": []any{
			map[string]any{"type": "command", "command": cmdString(cx, route)},
		},
	}
}

// ensureMap returns the sub-map at key, creating it if absent.
func ensureMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]any); ok {
			return sub
		}
	}
	sub := map[string]any{}
	m[key] = sub
	return sub
}

// patchJSON creates (or overwrites) the file at path with the result of patch applied to an empty map.
func patchJSON(path string, patch func(map[string]any)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	m := map[string]any{}
	patch(m)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

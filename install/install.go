// Package install writes per-agent hook configuration into each agent's settings
// file. It is the canonical installer consumers (e.g. the cx CLI) call to wire a
// hook binary into Claude Code, Cursor, Windsurf, Factory Droid, and Gemini CLI.
//
// Each InstallX function writes that agent's curated route set — the routes a
// consumer exposes as `<binary> hooks <route>` subcommands. The settings-file path,
// event key, and encoding style for every route come from the route Catalog
// (the single source of truth), so this package stays a thin, declarative wrapper.
package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
)

// CmdForFunc returns the shell command a given route should run. Consumers control
// how routes map to commands (e.g. "cx hooks <route>") by supplying this.
type CmdForFunc func(route string) string

// FormatCommand joins parts into a single space-separated command string, e.g.
// FormatCommand("/usr/bin/cx", "hooks", "claude-stop") == "/usr/bin/cx hooks claude-stop".
func FormatCommand(parts ...string) string { return strings.Join(parts, " ") }

// Curated per-agent route sets. A consumer's own command list (the routes it exposes
// as subcommands) must mirror these so every installed command resolves.
var (
	claudeRoutes = []string{
		"claude-stop", "claude-pre-tool-use", "claude-pre-file-write", "claude-user-prompt-submit",
	}
	cursorRoutes = []string{
		"cursor-stop", "cursor-before-shell", "cursor-before-mcp",
		"cursor-before-file-read", "cursor-after-file-edit", "cursor-before-submit-prompt",
	}
	windsurfRoutes = []string{
		"windsurf-pre-run-command", "windsurf-pre-mcp-tool-use", "windsurf-pre-user-prompt",
		"windsurf-pre-write-code", "windsurf-post-cascade-response",
	}
	droidRoutes = []string{
		"droid-stop", "droid-pre-tool-use", "droid-pre-file-write", "droid-user-prompt-submit",
	}
	geminiRoutes = []string{
		"gemini-before-agent", "gemini-before-tool", "gemini-before-file-tool", "gemini-after-agent",
	}
	copilotCLIRoutes = []string{
		"copilot-cli-stop", "copilot-cli-pre-tool-use", "copilot-cli-pre-file-write", "copilot-cli-user-prompt-submit",
	}
)

// InstallClaude writes Claude Code hook config (~/.claude/settings.json) under home.
func InstallClaude(home string, cmdFor CmdForFunc) error {
	return install(home, claudeRoutes, cmdFor)
}

// InstallCursor writes Cursor hook config (~/.cursor/hooks.json) under home.
func InstallCursor(home string, cmdFor CmdForFunc) error {
	return install(home, cursorRoutes, cmdFor)
}

// InstallWindsurf writes Windsurf Cascade hook config (~/.codeium/windsurf/hooks.json) under home.
func InstallWindsurf(home string, cmdFor CmdForFunc) error {
	return install(home, windsurfRoutes, cmdFor)
}

// InstallDroid writes Factory Droid hook config (~/.factory/settings.json) under home.
func InstallDroid(home string, cmdFor CmdForFunc) error {
	return install(home, droidRoutes, cmdFor)
}

// InstallGemini writes Gemini CLI hook config (~/.gemini/settings.json) under home.
func InstallGemini(home string, cmdFor CmdForFunc) error {
	return install(home, geminiRoutes, cmdFor)
}

// InstallCopilotCLI writes GitHub Copilot CLI hook config (~/.copilot/hooks/agenthooks.json) under home.
func InstallCopilotCLI(home string, cmdFor CmdForFunc) error {
	return install(home, copilotCLIRoutes, cmdFor)
}

// install groups the given routes by their settings file (from the Catalog) and
// writes each file once, preserving order.
func install(home string, routes []string, cmdFor CmdForFunc) error {
	byRoute := map[string]agenthooks.CatalogEntry{}
	for _, e := range agenthooks.Catalog {
		if _, seen := byRoute[e.Route]; !seen {
			byRoute[e.Route] = e
		}
	}

	type group struct {
		rel     string
		entries []agenthooks.CatalogEntry
	}
	groups := map[string]*group{}
	var order []string
	for _, r := range routes {
		e, ok := byRoute[r]
		if !ok {
			return fmt.Errorf("install: route %q is not in the catalog", r)
		}
		g := groups[e.SettingsRel]
		if g == nil {
			g = &group{rel: e.SettingsRel}
			groups[e.SettingsRel] = g
			order = append(order, e.SettingsRel)
		}
		g.entries = append(g.entries, e)
	}

	for _, rel := range order {
		g := groups[rel]
		path := filepath.Join(home, filepath.FromSlash(rel))
		if err := patchJSONFile(path, func(m map[string]any) {
			for _, e := range g.entries {
				writeHookEntry(m, e, cmdFor(e.Route))
			}
		}); err != nil {
			return fmt.Errorf("install %s: %w", rel, err)
		}
	}
	return nil
}

// writeHookEntry installs one catalog entry into the in-memory settings map per its
// platform's encoding style. Nested styles append to (and de-dup within) the event
// array so a user's own hooks are preserved; the flat style is single-command per key.
func writeHookEntry(m map[string]any, e agenthooks.CatalogEntry, cmd string) {
	switch e.Style {
	case agenthooks.StyleClaudeNested:
		hooks := ensureMap(m, "hooks")
		arr := toSlice(hooks[e.EventKey])
		if !containsCommand(arr, cmd) {
			hooks[e.EventKey] = append(arr, map[string]any{"type": "command", "command": cmd})
		}
	case agenthooks.StyleGeminiNested:
		hooks := ensureMap(m, "hooks")
		arr := toSlice(hooks[e.EventKey])
		if !containsCommand(arr, cmd) {
			hooks[e.EventKey] = append(arr, map[string]any{
				"matcher": "",
				"hooks":   []any{map[string]any{"type": "command", "command": cmd}},
			})
		}
	case agenthooks.StyleCopilotCLINested:
		if _, ok := m["version"]; !ok {
			m["version"] = 1
		}
		hooks := ensureMap(m, "hooks")
		arr := toSlice(hooks[e.EventKey])
		if !containsCommand(arr, cmd) {
			hooks[e.EventKey] = append(arr, map[string]any{"type": "command", "command": cmd})
		}
	case agenthooks.StyleFlatCommand:
		m[e.EventKey] = map[string]any{"command": cmd}
	}
}

// patchJSONFile reads path (creating it if absent), applies patch, and writes it
// back. If the existing file is non-empty but invalid JSON it aborts without writing
// (so user config is never clobbered) and backs the original up to <path>.bak.
func patchJSONFile(path string, patch func(map[string]any)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	m := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("refusing to overwrite %s: existing file is not valid JSON: %w", path, err)
		}
		if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
			return fmt.Errorf("writing backup %s.bak: %w", path, err)
		}
	}
	patch(m)
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

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

func toSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func containsCommand(arr []any, cmd string) bool {
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if m["command"] == cmd {
			return true
		}
		if inner, ok := m["hooks"].([]any); ok && containsCommand(inner, cmd) {
			return true
		}
	}
	return false
}

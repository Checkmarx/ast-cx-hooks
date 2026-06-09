// Package install provides shared primitives for writing agent hook
// configuration files. It is consumed by both the standalone agenthooks
// CLI (cmd/agenthooks) and by external installers such as the Checkmarx
// cx CLI's "hooks agenthooks install" command, so per-agent JSON shapes
// stay aligned across tools.
package install

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PatchJSON reads the existing JSON file at path (if any), applies patch to
// the parsed map, then writes the result back. Existing keys not touched by
// patch are preserved, so user settings (theme, keybindings, permissions,
// etc.) survive. The parent directory is created if absent.
func PatchJSON(path string, patch func(map[string]any)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	m := map[string]any{}
	if existing, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(existing, &m)
	}
	patch(m)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// EnsureMap returns the sub-map at key, creating it if absent. Useful when
// building nested JSON structures inside a PatchJSON callback.
func EnsureMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]any); ok {
			return sub
		}
	}
	sub := map[string]any{}
	m[key] = sub
	return sub
}

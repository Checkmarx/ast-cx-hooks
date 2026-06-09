package install

import "strings"

// CmdForFunc returns the shell command to invoke for a given route name.
// Per-agent installers call this for each hook event they wire so the caller
// controls how routes map to commands (e.g. "cx hooks <route>" for the ast-cli
// CLI, or "/path/to/hookbinary <route>" for a standalone hook binary).
type CmdForFunc func(route string) string

// FormatCommand formats a shell command suitable for an agent hook config:
// backslashes in the binary path are converted to forward slashes so the
// command survives bash -c evaluation on Windows (Git Bash / MSYS2 — which
// every supported agent's hook runner uses), and the binary is double-quoted
// if it contains spaces. Additional args are appended space-separated.
func FormatCommand(binary string, args ...string) string {
	binary = strings.ReplaceAll(binary, `\`, `/`)
	if strings.Contains(binary, " ") {
		binary = `"` + binary + `"`
	}
	if len(args) == 0 {
		return binary
	}
	return binary + " " + strings.Join(args, " ")
}

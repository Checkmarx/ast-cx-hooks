// Command agenthooks provides install, build, and init utilities for agent hooks.
//
// Usage:
//
//	agenthooks init [--dir <path>]   — scaffold a new hooks project
//	agenthooks install <binary>      — write hook configs pointing to <binary> into all agent settings files
//	agenthooks build                 — cross-compile your hook binary for all supported platforms
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/scaffold"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "init":
		scaffold.Run(os.Args[2:])
	case "install":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: agenthooks install <binary-path>")
			os.Exit(1)
		}
		if err := runInstall(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "install:", err)
			os.Exit(1)
		}
	case "build":
		if err := runBuild(); err != nil {
			fmt.Fprintln(os.Stderr, "build:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "agenthooks — hook configuration tool")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  init [--dir <path>]   Scaffold a new hooks project (default: current directory)")
	fmt.Fprintln(os.Stderr, "  install <binary>      Write hook configs for all agents pointing to <binary>")
	fmt.Fprintln(os.Stderr, "  build                 Cross-compile hook binary for all platforms")
}

// =============================================================================
// install
// =============================================================================

func runInstall(binaryPath string) error {
	abs, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}

	installFns := []struct {
		name string
		fn   func(string, string) error
	}{
		{"Claude Code", installClaude},
		{"Cursor", installCursor},
		{"Windsurf Cascade", installWindsurf},
		{"Factory Droid", installDroid},
	}

	for _, item := range installFns {
		if err := item.fn(home, abs); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", item.name, err)
		} else {
			fmt.Printf("✓ %s configured\n", item.name)
		}
	}
	return nil
}

// installClaude writes hook configuration to ~/.claude/settings.json.
func installClaude(home, binary string) error {
	path := filepath.Join(home, ".claude", "settings.json")
	return patchJSONFile(path, func(m map[string]any) {
		hooks := ensureMap(m, "hooks")
		hooks["Stop"] = hookEntries(binary, "claude-stop")
		hooks["PreToolUse"] = hookEntries(binary, "claude-pre-tool-use")
		hooks["PostToolUse"] = hookEntries(binary, "claude-after-file-write")
		hooks["UserPromptSubmit"] = hookEntries(binary, "claude-user-prompt-submit")
	})
}

// installCursor writes hook configuration to ~/.cursor/hooks.json.
func installCursor(home, binary string) error {
	path := filepath.Join(home, ".cursor", "hooks.json")
	return patchJSONFile(path, func(m map[string]any) {
		m["stop"] = cursorHook(binary, "cursor-stop")
		m["beforeShellExecution"] = cursorHook(binary, "cursor-before-shell")
		m["beforeMCPExecution"] = cursorHook(binary, "cursor-before-mcp")
		m["afterFileEdit"] = cursorHook(binary, "cursor-after-file-edit")
		m["beforeSubmitPrompt"] = cursorHook(binary, "cursor-before-submit-prompt")
	})
}

// installWindsurf writes hook configuration to ~/.codeium/windsurf/hooks.json.
func installWindsurf(home, binary string) error {
	path := filepath.Join(home, ".codeium", "windsurf", "hooks.json")
	return patchJSONFile(path, func(m map[string]any) {
		m["pre_run_command"] = windsurfHook(binary, "windsurf-pre-run-command")
		m["pre_mcp_tool_use"] = windsurfHook(binary, "windsurf-pre-mcp-tool-use")
		m["pre_user_prompt"] = windsurfHook(binary, "windsurf-pre-user-prompt")
		m["post_write_code"] = windsurfHook(binary, "windsurf-post-write-code")
		m["post_cascade_response"] = windsurfHook(binary, "windsurf-post-cascade-response")
	})
}

// installDroid writes hook configuration to ~/.factory/settings.json.
func installDroid(home, binary string) error {
	path := filepath.Join(home, ".factory", "settings.json")
	return patchJSONFile(path, func(m map[string]any) {
		hooks := ensureMap(m, "hooks")
		hooks["Stop"] = hookEntries(binary, "droid-stop")
		hooks["PreToolUse"] = hookEntries(binary, "droid-pre-tool-use")
		hooks["PostToolUse"] = hookEntries(binary, "droid-after-file-write")
		hooks["UserPromptSubmit"] = hookEntries(binary, "droid-user-prompt-submit")
	})
}

// hookEntries returns a Claude/Droid-style hook entry array.
func hookEntries(binary, subcmd string) []map[string]any {
	return []map[string]any{
		{"type": "command", "command": binary + " " + subcmd},
	}
}

// cursorHook returns a Cursor-style hook entry.
func cursorHook(binary, subcmd string) map[string]any {
	return map[string]any{"command": binary + " " + subcmd}
}

// windsurfHook returns a Windsurf-style hook entry.
func windsurfHook(binary, subcmd string) map[string]any {
	return map[string]any{"command": binary + " " + subcmd}
}

// patchJSONFile reads a JSON file (creating it if absent), applies patch, and writes it back.
func patchJSONFile(path string, patch func(map[string]any)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	m := map[string]any{}
	if data, err := os.ReadFile(path); err == nil { // #nosec G304
		_ = json.Unmarshal(data, &m)
	}
	patch(m)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600) // #nosec
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

// =============================================================================
// build
// =============================================================================

var buildTargets = []struct{ goos, goarch string }{
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"windows", "amd64"},
	{"windows", "arm64"},
}

func runBuild() error {
	// Determine the package to build (default: current directory).
	pkg := "."
	if len(os.Args) > 2 {
		pkg = os.Args[2]
	}

	outDir := "dist"
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return err
	}

	baseName := filepath.Base(pkg)
	if baseName == "." {
		cwd, _ := os.Getwd()
		baseName = filepath.Base(cwd)
	}

	for _, t := range buildTargets {
		name := fmt.Sprintf("%s-%s-%s", baseName, t.goos, t.goarch)
		if t.goos == "windows" {
			name += ".exe"
		}
		out := filepath.Join(outDir, name)

		cmd := exec.Command("go", "build", "-o", out, pkg) // #nosec G702,G204
		cmd.Env = append(os.Environ(),
			"GOOS="+t.goos,
			"GOARCH="+t.goarch,
			"CGO_ENABLED=0",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Printf("building %s/%s → %s\n", t.goos, t.goarch, out)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed for %s/%s: %w", t.goos, t.goarch, err)
		}
	}

	fmt.Printf("build complete — binaries in %s/\n", outDir)
	return nil
}

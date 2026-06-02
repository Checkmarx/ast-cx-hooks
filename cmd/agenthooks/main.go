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

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
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

	// Group catalog entries by settings file so each file is written once,
	// preserving registration order for stable output.
	type group struct {
		entries []agenthooks.CatalogEntry
	}
	groups := map[string]*group{}
	var order []string
	for _, e := range agenthooks.Catalog {
		g := groups[e.SettingsRel]
		if g == nil {
			g = &group{}
			groups[e.SettingsRel] = g
			order = append(order, e.SettingsRel)
		}
		g.entries = append(g.entries, e)
	}

	for _, rel := range order {
		g := groups[rel]
		path := filepath.Join(home, filepath.FromSlash(rel))
		err := patchJSONFile(path, func(m map[string]any) {
			for _, e := range g.entries {
				writeHookEntry(m, e, abs)
			}
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", rel, err)
		} else {
			fmt.Printf("✓ %s (%d hooks) → %s\n", g.entries[0].Agent, len(g.entries), rel)
		}
	}
	fmt.Println("note: VS Code Copilot hooks are project-scoped (.github/hooks) — see README for manual setup")
	return nil
}

// writeHookEntry installs a single catalog entry into the in-memory settings map
// according to its platform's encoding style.
func writeHookEntry(m map[string]any, e agenthooks.CatalogEntry, binary string) {
	cmd := binary + " " + e.Route
	switch e.Style {
	case agenthooks.StyleClaudeNested:
		ensureMap(m, "hooks")[e.EventKey] = []map[string]any{
			{"type": "command", "command": cmd},
		}
	case agenthooks.StyleGeminiNested:
		ensureMap(m, "hooks")[e.EventKey] = []map[string]any{
			{"matcher": "", "hooks": []map[string]any{{"type": "command", "command": cmd}}},
		}
	case agenthooks.StyleFlatCommand:
		m[e.EventKey] = map[string]any{"command": cmd}
	}
}

// patchJSONFile reads a JSON file (creating it if absent), applies patch, and writes it back.
func patchJSONFile(path string, patch func(map[string]any)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	m := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &m) //nolint:errcheck
	}
	patch(m)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
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
	if err := os.MkdirAll(outDir, 0o755); err != nil {
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

		cmd := exec.Command("go", "build", "-o", out, pkg)
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

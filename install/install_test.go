package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// cmdFor maps a route to "/bin/cx hooks <route>", mirroring how a consumer CLI
// wires its own routes into agent config files.
func testCmdFor(route string) string { return FormatCommand("/bin/cx", "hooks", route) }

// TestInstallCopilotCLI verifies the GitHub Copilot CLI installer writes the
// expected hooks file (~/.copilot/hooks/agenthooks.json) with the curated route
// set encoded in the Copilot CLI style ({"version":1,"hooks":{EventKey:[...]}}).
func TestInstallCopilotCLI(t *testing.T) {
	home := t.TempDir()
	if err := InstallCopilotCLI(home, testCmdFor); err != nil {
		t.Fatalf("InstallCopilotCLI: %v", err)
	}

	path := filepath.Join(home, ".copilot", "hooks", "agenthooks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected config at %s: %v", path, err)
	}

	var cfg struct {
		Version int `json:"version"`
		Hooks   map[string][]struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("config is not valid JSON: %v\n%s", err, data)
	}

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}

	// The curated set maps onto three event keys; pre-tool-use and pre-file-write
	// both land under PreToolUse, so it carries two commands.
	wantCommands := map[string]string{
		"Stop":             "/bin/cx hooks copilot-cli-stop",
		"UserPromptSubmit": "/bin/cx hooks copilot-cli-user-prompt-submit",
	}
	for key, want := range wantCommands {
		entries, ok := cfg.Hooks[key]
		if !ok || len(entries) != 1 {
			t.Fatalf("hooks[%q] = %+v, want exactly one entry", key, entries)
		}
		if entries[0].Type != "command" || entries[0].Command != want {
			t.Errorf("hooks[%q][0] = %+v, want command %q", key, entries[0], want)
		}
	}

	pre := cfg.Hooks["PreToolUse"]
	if len(pre) != 2 {
		t.Fatalf("hooks[PreToolUse] = %+v, want two commands (pre-tool-use + pre-file-write)", pre)
	}
	got := map[string]bool{pre[0].Command: true, pre[1].Command: true}
	for _, want := range []string{
		"/bin/cx hooks copilot-cli-pre-tool-use",
		"/bin/cx hooks copilot-cli-pre-file-write",
	} {
		if !got[want] {
			t.Errorf("hooks[PreToolUse] missing %q; got %+v", want, pre)
		}
	}
}

// TestInstallCopilotCLIIdempotent verifies re-installing does not duplicate entries.
func TestInstallCopilotCLIIdempotent(t *testing.T) {
	home := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := InstallCopilotCLI(home, testCmdFor); err != nil {
			t.Fatalf("InstallCopilotCLI (pass %d): %v", i, err)
		}
	}

	path := filepath.Join(home, ".copilot", "hooks", "agenthooks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Hooks map[string][]any `json:"hooks"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if n := len(cfg.Hooks["PreToolUse"]); n != 2 {
		t.Errorf("after re-install, hooks[PreToolUse] has %d entries, want 2 (no duplicates)", n)
	}
	if n := len(cfg.Hooks["Stop"]); n != 1 {
		t.Errorf("after re-install, hooks[Stop] has %d entries, want 1 (no duplicates)", n)
	}
}

// TestInstallCodex verifies the OpenAI Codex CLI installer writes the expected
// hooks file (~/.codex/hooks.json) with the curated route set encoded in the
// Gemini-style nested form ({"hooks":{EventKey:[{"matcher":"","hooks":[...]}]}}).
func TestInstallCodex(t *testing.T) {
	home := t.TempDir()
	if err := InstallCodex(home, testCmdFor); err != nil {
		t.Fatalf("InstallCodex: %v", err)
	}

	path := filepath.Join(home, ".codex", "hooks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected config at %s: %v", path, err)
	}

	type entry struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		} `json:"hooks"`
	}
	var cfg struct {
		Hooks map[string][]entry `json:"hooks"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("config is not valid JSON: %v\n%s", err, data)
	}

	wantCommands := map[string]string{
		"Stop":             "/bin/cx hooks codex-stop",
		"UserPromptSubmit": "/bin/cx hooks codex-user-prompt-submit",
	}
	for key, want := range wantCommands {
		entries, ok := cfg.Hooks[key]
		if !ok || len(entries) != 1 || len(entries[0].Hooks) != 1 {
			t.Fatalf("hooks[%q] = %+v, want exactly one nested entry", key, entries)
		}
		if entries[0].Hooks[0].Type != "command" || entries[0].Hooks[0].Command != want {
			t.Errorf("hooks[%q][0] = %+v, want command %q", key, entries[0].Hooks[0], want)
		}
	}

	pre := cfg.Hooks["PreToolUse"]
	if len(pre) != 2 {
		t.Fatalf("hooks[PreToolUse] = %+v, want two entries (pre-tool-use + pre-file-write)", pre)
	}
	got := map[string]bool{}
	for _, e := range pre {
		for _, h := range e.Hooks {
			got[h.Command] = true
		}
	}
	for _, want := range []string{
		"/bin/cx hooks codex-pre-tool-use",
		"/bin/cx hooks codex-pre-file-write",
	} {
		if !got[want] {
			t.Errorf("hooks[PreToolUse] missing %q; got %+v", want, pre)
		}
	}
}

// TestInstallCodexIdempotent verifies re-installing does not duplicate entries.
func TestInstallCodexIdempotent(t *testing.T) {
	home := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := InstallCodex(home, testCmdFor); err != nil {
			t.Fatalf("InstallCodex (pass %d): %v", i, err)
		}
	}

	path := filepath.Join(home, ".codex", "hooks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Hooks map[string][]any `json:"hooks"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if n := len(cfg.Hooks["PreToolUse"]); n != 2 {
		t.Errorf("after re-install, hooks[PreToolUse] has %d entries, want 2 (no duplicates)", n)
	}
	if n := len(cfg.Hooks["Stop"]); n != 1 {
		t.Errorf("after re-install, hooks[Stop] has %d entries, want 1 (no duplicates)", n)
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
)

// TestWriteHookEntryShapes pins the on-disk JSON shape each HookStyle produces.
func TestWriteHookEntryShapes(t *testing.T) {
	cases := []struct {
		name  string
		entry agenthooks.CatalogEntry
		want  string
	}{
		{
			name:  "claude-nested",
			entry: agenthooks.CatalogEntry{Route: "claude-stop", EventKey: "Stop", Style: agenthooks.StyleClaudeNested},
			want:  `{"hooks":{"Stop":[{"command":"/bin/myhook claude-stop","type":"command"}]}}`,
		},
		{
			name:  "gemini-nested",
			entry: agenthooks.CatalogEntry{Route: "gemini-before-tool", EventKey: "BeforeTool", Style: agenthooks.StyleGeminiNested},
			want:  `{"hooks":{"BeforeTool":[{"hooks":[{"command":"/bin/myhook gemini-before-tool","type":"command"}],"matcher":""}]}}`,
		},
		{
			name:  "flat-command",
			entry: agenthooks.CatalogEntry{Route: "cursor-stop", EventKey: "stop", Style: agenthooks.StyleFlatCommand},
			want:  `{"stop":{"command":"/bin/myhook cursor-stop"}}`,
		},
		{
			name:  "copilot-cli-nested",
			entry: agenthooks.CatalogEntry{Route: "copilot-cli-stop", EventKey: "Stop", Style: agenthooks.StyleCopilotCLINested},
			want:  `{"hooks":{"Stop":[{"command":"/bin/myhook copilot-cli-stop","type":"command"}]},"version":1}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string]any{}
			writeHookEntry(m, tc.entry, "/bin/myhook")
			got, _ := json.Marshal(m)
			if string(got) != tc.want {
				t.Fatalf("shape mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// TestWriteHookEntryMergesAndDedups verifies that installing preserves a user's
// pre-existing entry under the same event and does not duplicate on re-install.
func TestWriteHookEntryMergesAndDedups(t *testing.T) {
	m := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{"type": "command", "command": "/user/own-hook"},
			},
		},
	}
	e := agenthooks.CatalogEntry{Route: "claude-pre-tool-use", EventKey: "PreToolUse", Style: agenthooks.StyleClaudeNested}

	writeHookEntry(m, e, "/bin/myhook") // first install
	writeHookEntry(m, e, "/bin/myhook") // re-install must not duplicate

	arr := m["hooks"].(map[string]any)["PreToolUse"].([]any)
	if len(arr) != 2 {
		t.Fatalf("expected user entry preserved + ours appended once, got %d entries: %+v", len(arr), arr)
	}
	js, _ := json.Marshal(m)
	if !strings.Contains(string(js), "/user/own-hook") {
		t.Fatalf("user's pre-existing hook was dropped: %s", js)
	}
	if !strings.Contains(string(js), "/bin/myhook claude-pre-tool-use") {
		t.Fatalf("our hook was not added: %s", js)
	}
}

// TestPatchJSONFilePreservesUnrelatedKeys verifies install keeps unrelated config.
func TestPatchJSONFilePreservesUnrelatedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"dark","hooks":{"Other":[1]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := patchJSONFile(path, func(m map[string]any) {
		writeHookEntry(m, agenthooks.CatalogEntry{Route: "claude-stop", EventKey: "Stop", Style: agenthooks.StyleClaudeNested}, "/bin/myhook")
	})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	data, _ := os.ReadFile(path)
	js := string(data)
	for _, want := range []string{`"theme": "dark"`, `"Other"`, `"Stop"`, "claude-stop"} {
		if !strings.Contains(js, want) {
			t.Fatalf("result missing %q:\n%s", want, js)
		}
	}
}

// TestPatchJSONFileAbortsOnInvalidJSON verifies a malformed settings file is left
// untouched rather than silently overwritten.
func TestPatchJSONFileAbortsOnInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	original := "{ this is not valid json // comment\n}"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	err := patchJSONFile(path, func(m map[string]any) { m["x"] = 1 })
	if err == nil {
		t.Fatal("expected patchJSONFile to abort on invalid JSON, got nil error")
	}
	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Fatalf("original file was modified despite abort:\n%s", data)
	}
}

// TestPatchJSONFileBacksUp verifies a .bak is written before modifying an existing file.
func TestPatchJSONFileBacksUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"keep":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := patchJSONFile(path, func(m map[string]any) { m["added"] = true }); err != nil {
		t.Fatalf("patch: %v", err)
	}
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
	if string(bak) != `{"keep":true}` {
		t.Fatalf("backup should hold the original content, got: %s", bak)
	}
}

// TestEveryCatalogEntryInstallsToValidJSON exercises the full Catalog through the
// installer and asserts every produced settings file is valid JSON.
func TestEveryCatalogEntryInstallsToValidJSON(t *testing.T) {
	seenStyles := map[agenthooks.HookStyle]bool{}
	for _, e := range agenthooks.Catalog {
		m := map[string]any{}
		writeHookEntry(m, e, "/bin/myhook")
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("catalog entry %q produced unmarshalable map: %v", e.Route, err)
		}
		var back map[string]any
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("catalog entry %q produced invalid JSON: %v", e.Route, err)
		}
		seenStyles[e.Style] = true
	}
	for _, s := range []agenthooks.HookStyle{agenthooks.StyleClaudeNested, agenthooks.StyleGeminiNested, agenthooks.StyleFlatCommand, agenthooks.StyleCopilotCLINested} {
		if !seenStyles[s] {
			t.Errorf("no catalog entry exercises HookStyle %q", s)
		}
	}
}

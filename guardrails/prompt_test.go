package guardrails

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// resolveReferencedFile is the resolver behind ScanReferencedFiles. We exercise
// it directly because the scanner integration is unchanged — only the resolver
// logic shifted from "literal stat" to "literal stat + glob fallback".

func TestResolveReferencedFile_LiteralAbsoluteHit(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.yml")
	mustWrite(t, target, "k: v")

	got := resolveReferencedFile(target, nil)
	if len(got) != 1 || got[0] != target {
		t.Fatalf("expected [%q], got %v", target, got)
	}
}

func TestResolveReferencedFile_GlobFallbackFindsSibling(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "application-jira.yml"), "k: v")

	typed := filepath.Join(dir, "application-jira") // no extension
	got := resolveReferencedFile(typed, nil)

	if len(got) != 1 || filepath.Base(got[0]) != "application-jira.yml" {
		t.Fatalf("expected glob fallback to find application-jira.yml, got %v", got)
	}
}

func TestResolveReferencedFile_GlobFallbackNoSibling(t *testing.T) {
	dir := t.TempDir()
	// parent exists but nothing matches the prefix
	typed := filepath.Join(dir, "application-jira")
	if got := resolveReferencedFile(typed, nil); got != nil {
		t.Fatalf("expected nil when nothing matches, got %v", got)
	}
}

func TestResolveReferencedFile_GlobFallbackBailsOnTooManyMatches(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i <= maxGlobFallbackMatches; i++ {
		mustWrite(t, filepath.Join(dir, "common-prefix-"+itoa(i)+".log"), "x")
	}

	typed := filepath.Join(dir, "common-prefix")
	if got := resolveReferencedFile(typed, nil); got != nil {
		t.Fatalf("expected nil when match count exceeds cap, got %d entries", len(got))
	}
}

func TestResolveReferencedFile_TypedPathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "secrets")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mustWrite(t, filepath.Join(subdir, "creds.yml"), "k: v")

	if got := resolveReferencedFile(subdir, nil); got != nil {
		t.Fatalf("expected nil for directory reference, got %v", got)
	}
}

func TestResolveReferencedFile_RelativePathResolvesAgainstWorkspaceRoot(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "application-jira.yml"), "k: v")

	got := resolveReferencedFile("application-jira", []string{dir})
	if len(got) != 1 || filepath.Base(got[0]) != "application-jira.yml" {
		t.Fatalf("expected glob fallback under workspace root to find application-jira.yml, got %v", got)
	}
}

func TestResolveReferencedFile_RelativeStopsAtFirstMatchingRoot(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	mustWrite(t, filepath.Join(rootA, "config.yml"), "a")
	mustWrite(t, filepath.Join(rootB, "config.yml"), "b")

	got := resolveReferencedFile("config.yml", []string{rootA, rootB})
	if len(got) != 1 || filepath.Dir(got[0]) != rootA {
		t.Fatalf("expected resolution to stop at rootA, got %v", got)
	}
}

func TestResolveReferencedFile_CursorStyleWindowsRootNormalised(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Cursor /c:/ root form is Windows-specific")
	}
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "application-jira.yml"), "k: v")

	// Cursor reports Windows roots as "/c:/foo"; NormalizeWorkspaceRoot strips
	// the leading slash. Confirm the resolver still finds the file via glob.
	cursorRoot := "/" + filepath.ToSlash(dir)
	got := resolveReferencedFile("application-jira", []string{cursorRoot})
	if len(got) != 1 || filepath.Base(got[0]) != "application-jira.yml" {
		t.Fatalf("expected glob fallback under Cursor-style root, got %v", got)
	}
}

func TestResolveReferencedFile_GlobMatchesMixedRegularAndDir(t *testing.T) {
	// A directory whose name shares the prefix must not be returned as a file.
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "app.yml"), "k: v")
	if err := os.Mkdir(filepath.Join(dir, "app-data"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	typed := filepath.Join(dir, "app")
	got := resolveReferencedFile(typed, nil)
	sort.Strings(got)
	if len(got) != 1 || filepath.Base(got[0]) != "app.yml" {
		t.Fatalf("expected only the regular file, got %v", got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

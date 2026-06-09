package scaffold

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunGeneratesValidProject scaffolds into a temp dir and verifies the
// expected files exist and every generated *.go file parses cleanly. Parsing
// catches a generated template that no longer compiles against the API.
func TestRunGeneratesValidProject(t *testing.T) {
	dir := t.TempDir()

	Run([]string{"-dir", dir})

	wantFiles := []string{"main.go", "README.md", ".gitignore", "policy.json"}
	for _, name := range wantFiles {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected generated file %q: %v", name, err)
		}
	}

	// Parse every generated Go file; fail on any parse error.
	goFiles, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(goFiles) == 0 {
		t.Fatal("expected at least one generated *.go file, found none")
	}
	for _, p := range goFiles {
		if _, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.AllErrors); err != nil {
			t.Errorf("generated Go file %q does not parse: %v", p, err)
		}
	}
}

// TestRunRefusesToOverwrite verifies that Run refuses to overwrite an existing
// target file. Run signals this via os.Exit(1), so it is exercised in a child
// process (the standard Go pattern for testing os.Exit). The child runs the
// helper below, guarded by the SCAFFOLD_OVERWRITE_HELPER env var.
func TestRunRefusesToOverwrite(t *testing.T) {
	if os.Getenv("SCAFFOLD_OVERWRITE_HELPER") == "1" {
		// Child process: pre-create one target file, then attempt to scaffold.
		dir := os.Getenv("SCAFFOLD_OVERWRITE_DIR")
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("// pre-existing\n"), 0o644); err != nil {
			t.Fatalf("child: pre-create main.go: %v", err)
		}
		Run([]string{"-dir", dir}) // expected to call os.Exit(1)
		return
	}

	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run", "TestRunRefusesToOverwrite")
	cmd.Env = append(os.Environ(),
		"SCAFFOLD_OVERWRITE_HELPER=1",
		"SCAFFOLD_OVERWRITE_DIR="+dir,
	)
	out, err := cmd.CombinedOutput()

	// The child must exit non-zero because the target already exists.
	if err == nil {
		t.Fatalf("expected Run to exit non-zero when target exists; child output:\n%s", out)
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.Success() {
		t.Fatalf("expected a non-zero ExitError, got %v; output:\n%s", err, out)
	}
	if !strings.Contains(string(out), "already exists") {
		t.Errorf("expected 'already exists' message in child output, got:\n%s", out)
	}
}

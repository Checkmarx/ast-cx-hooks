// Package scaffold generates a starter hooks project from built-in templates.
package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Run scaffolds a new agenthooks project into dir (default ".").
// It accepts an optional "-dir <path>" flag as args.
func Run(args []string) {
	dir := "."
	for i := 0; i < len(args); i++ {
		if (args[i] == "-dir" || args[i] == "--dir") && i+1 < len(args) {
			dir = args[i+1]
			i++
		}
	}

	files := templateFiles()

	// Refuse to overwrite any existing file.
	for _, f := range files {
		out := filepath.Join(dir, filepath.FromSlash(f.path))
		if _, err := os.Stat(out); err == nil {
			fmt.Fprintf(os.Stderr, "error: %s already exists\n", out)
			os.Exit(1)
		} else if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "error checking %s: %v\n", out, err)
			os.Exit(1)
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", dir, err)
		os.Exit(1)
	}

	for _, f := range files {
		out := filepath.Join(dir, filepath.FromSlash(f.path))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error creating directory for %s: %v\n", out, err)
			os.Exit(1)
		}
		if err := os.WriteFile(out, []byte(f.content), f.mode); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", out, err)
			os.Exit(1)
		}
		fmt.Printf("  created %s\n", out)
	}

	fmt.Printf("\nScaffolded agenthooks project in %s\n", dir)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. cd", dir)
	fmt.Println("  2. go mod init <your-module-path>")
	fmt.Println("  3. go get github.com/CheckmarxDev/ast-cx-hooks@latest")
	fmt.Println("  4. go mod tidy")
	fmt.Println("  5. go build -o my-hooks .")
	fmt.Println("  6. agenthooks install ./my-hooks")
}

type templateFile struct {
	path    string
	content string
	mode    os.FileMode
}

func templateFiles() []templateFile {
	return []templateFile{
		{path: "main.go", content: tmplMainGo, mode: 0o644},
		{path: "README.md", content: tmplReadme, mode: 0o644},
		{path: ".gitignore", content: tmplGitignore, mode: 0o644},
		{path: "policy.json", content: tmplPolicyJSON, mode: 0o644},
	}
}

# Technology Stack

**Analysis Date:** 2026-04-28

## Languages

**Primary:**
- Go 1.21 - All core functionality, tests, CLI tools, and platform-specific packages

## Runtime

**Environment:**
- Go 1.21+ runtime

**Package Manager:**
- Go modules (go mod)
- Lockfile: Not present (minimal dependencies)

## Frameworks

**Core:**
- Standard library only (`encoding/json`, `os`, `fmt`, `bufio`, `path/filepath`, `errors`) - No external framework dependencies

**Testing:**
- Go testing package (built-in) - Located in `agenthooks_test.go`, `claude/responses_test.go`, `cursor/responses_test.go`, `internal/codec/codec_test.go`

**Build/Dev:**
- Go build system - Standard `go build` and cross-compilation via `go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks build`

## Key Dependencies

**Critical:**
- None - Project uses only Go standard library. Zero external dependencies.

**Internal Packages:**
- `github.com/CheckmarxDev/ast-cx-hooks/internal/codec` - JSON serialization between hooks and agents
- `github.com/CheckmarxDev/ast-cx-hooks/internal/scaffold` - Project scaffolding generator

## Configuration

**Environment:**
- No environment variables required for core framework
- Hook binaries receive context via stdin (JSON) and respond via stdout (JSON)
- Agent configuration files store hook binary paths:
  - Claude Code: `~/.claude/settings.json`
  - Cursor: `~/.cursor/hooks.json`
  - Windsurf Cascade: `~/.codeium/windsurf/hooks.json`
  - Factory Droid: `~/.factory/settings.json`
  - Gemini CLI: `~/.gemini/settings.json`

**Build:**
- `go.mod` - Go module definition (`module github.com/CheckmarxDev/ast-cx-hooks` with Go 1.21)
- No Makefile, Taskfile, or build configuration files
- Cross-compilation targets: macOS (amd64, arm64), Linux (amd64, arm64), Windows (amd64, arm64)

## Platform Requirements

**Development:**
- Go 1.21 or higher
- Unix-like shell (bash) for build/install scripts
- Cross-platform: builds run on Linux, macOS, Windows

**Production:**
- Hook binaries compile to standalone executables
- Required for: Claude Code, Cursor IDE, Windsurf Cascade, Factory Droid, Gemini CLI
- No runtime dependencies; binaries are self-contained

## Supported AI Agents

The framework explicitly supports:
- Claude Code (Anthropic)
- Cursor IDE
- Windsurf Cascade (Codeium)
- Factory Droid
- Gemini CLI (Google)

---

*Stack analysis: 2026-04-28*

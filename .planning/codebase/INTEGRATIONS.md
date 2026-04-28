# External Integrations

**Analysis Date:** 2026-04-28

## APIs & External Services

**AI Coding Agent Platforms:**
- Claude Code - Anthropic's code assistant
  - Hook event format: JSON stdin/stdout
  - Configuration: `~/.claude/settings.json`
  - Routes: `claude-pre-tool-use`, `claude-post-tool-use`, `claude-stop`, `claude-user-prompt-submit`

- Cursor IDE - Cursor code editor with AI
  - Hook event format: JSON stdin/stdout
  - Configuration: `~/.cursor/hooks.json`
  - Routes: `cursor-before-shell-execution`, `cursor-after-file-edit`, `cursor-stop`, `cursor-before-submit-prompt`
  - Note: Cursor on Windows prepends UTF-8 BOM to JSON payloads

- Windsurf Cascade - Codeium's agentic IDE
  - Hook event format: JSON stdin with exit code 2 to block
  - Configuration: `~/.codeium/windsurf/hooks.json`
  - Routes: `windsurf-pre-run-command`, `windsurf-post-write-code`, `windsurf-post-cascade-response`, `windsurf-pre-user-prompt`
  - Note: Pre-hooks block via exit code 2; post-hooks are fire-and-forget

- Factory Droid - AI agent framework
  - Hook event format: JSON stdin/stdout
  - Configuration: `~/.factory/settings.json`
  - Routes: `droid-pre-tool-use`, `droid-post-tool-use`, `droid-stop`, `droid-user-prompt-submit`

- Gemini CLI - Google's command-line agent
  - Hook event format: JSON stdin with exit code 2 to block
  - Configuration: `~/.gemini/settings.json` (not auto-configured by install command)
  - Routes: `gemini-before-tool`, `gemini-after-tool`, `gemini-before-agent`, `gemini-after-agent`

## Data Storage

**Databases:**
- None - Framework is stateless

**File Storage:**
- Local filesystem only
  - Agent settings files: home directory (`~/.claude/`, `~/.cursor/`, etc.)
  - Hook binary location: configurable (typically in `~/.claude/hooks/`, etc.)

**Caching:**
- None

## Authentication & Identity

**Auth Provider:**
- Custom - Each AI agent manages its own authentication
- Framework is not responsible for authentication
- Hook binaries run in the user's shell context with user privileges
- Session/conversation IDs passed via hook events for tracking

## Monitoring & Observability

**Error Tracking:**
- None - Framework logs errors to stderr via `os.Exit(2)` for blocking errors

**Logs:**
- stderr output: Used for error reporting and blocking messages
- stdout: Reserved for JSON response payload (binary protocol)
- Hook events include `session_id`, `conversation_id`, or `trajectory_id` for tracing

## CI/CD & Deployment

**Hosting:**
- GitHub repository: `github.com/CheckmarxDev/ast-cx-hooks`
- Releases: GitHub Releases with semantic versioning

**CI Pipeline:**
- GitHub Actions (`.github/workflows/Hooks-Release.yml`)
  - Trigger: Manual workflow dispatch with version input
  - Steps:
    - Checkout code (actions/checkout)
    - Install Go (actions/setup-go) via go.mod version
    - Validate version format (semantic: v1.2.3)
    - Bump version and create git tag (mathieudutour/github-tag-action)
    - Verify Go module: `go list -m all`, `go mod tidy`, `go mod verify`
    - Create GitHub Release (softprops/action-gh-release)

**Cross-Compilation:**
- Supported platforms: macOS (amd64, arm64), Linux (amd64, arm64), Windows (amd64, arm64)
- Build command: `go run github.com/CheckmarxDev/ast-cx-hooks/cmd/agenthooks build`

## Environment Configuration

**Required env vars:**
- None for framework itself
- Hook users may define custom env vars in agent settings

**Secrets location:**
- Agent settings files (`~/.claude/settings.json`, `~/.cursor/hooks.json`, etc.) may contain binary paths
- No secrets managed by framework

## Webhooks & Callbacks

**Incoming:**
- None - Framework is not a server

**Outgoing:**
- Hook binaries do not initiate outbound connections
- All integration is via agent-invoked stdin/stdout protocol

## Platform-Specific Behaviors

**Windows (Cursor only):**
- Cursor on Windows prepends UTF-8 BOM (`0xEF 0xBB 0xBF`) to JSON stdin
- Handled by codec: `internal/codec/codec.go` DecodeStdin function strips BOM before parsing

**Shell Integration:**
- Hook binaries receive current working directory (`cwd`) in events
- Can be used to make location-aware hook decisions
- Supported on Claude Code and Factory Droid

**Exit Codes:**
- Exit 0: Success (normal completion or graceful failure)
- Exit 1: Handler not found / registration error
- Exit 2: Blocking error (used by Windsurf, Gemini to block pending actions)

---

*Integration audit: 2026-04-28*

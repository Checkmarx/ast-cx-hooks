package scaffold

const tmplMainGo = `package main

import (
	"strings"

	hooks "github.com/Checkmarx/ast-cx-hooks"
)

func main() {
	// Register handlers (all optional — only register what you need).
	// hooks.WhenAgentIdle(handleIdle)
	// hooks.BeforeToolCall(handleTool)
	// hooks.AfterFileWrite(handleFileWrite)
	hooks.BeforePrompt(handlePrompt)

	hooks.Dispatch()
}

// handlePrompt blocks prompts that contain secrets or credentials.
func handlePrompt(ev hooks.PromptEvent) hooks.PromptVerdict {
	lower := strings.ToLower(ev.Text)

	// NOTE: these substring patterns are ILLUSTRATIVE examples, not a robust
	// control. They are easily bypassed via whitespace, encoding, or indirection.
	// Real policies should use allowlists, structured parsing, and input
	// normalization (defense in depth), not naive substring blocklists.
	for _, pat := range []string{
		"api_key=", "apikey=",
		"password=", "passwd=",
		"aws_access_key_id", "aws_secret_access_key",
		"bearer ", "token=",
		"-----begin rsa", "-----begin openssh", "private_key",
	} {
		if strings.Contains(lower, pat) {
			return hooks.RejectPrompt("Secret detected in prompt — remove credentials before sending.")
		}
	}

	// Well-known token prefixes (case-sensitive).
	for _, prefix := range []string{"ghp_", "gho_", "sk-", "xoxb-", "xoxp-"} {
		if strings.Contains(ev.Text, prefix) {
			return hooks.RejectPrompt("API token detected — use environment variables instead.")
		}
	}

	return hooks.AcceptPrompt()
}
`

const tmplReadme = `# My agenthooks Project

One hooks binary for Claude Code, Cursor, Windsurf Cascade, Factory Droid, and Gemini CLI.

## Quick Start

` + "```" + `bash
# 1. Initialize your Go module
go mod init github.com/your-org/your-hooks

# 2. Add the dependency
go get github.com/Checkmarx/ast-cx-hooks@latest
go mod tidy

# 3. Build your binary
go build -o my-hooks .

# 4. Install hooks into all five agent configs
agenthooks install ./my-hooks
` + "```" + `

## Unified Handlers

| Handler | Purpose | Can Block? |
|---|---|---|
| ` + "`WhenAgentIdle`" + ` | Control when the agent is allowed to finish | Claude / Cursor / Droid |
| ` + "`BeforeToolCall`" + ` | Gate shell commands and MCP tool invocations | All five agents |
| ` + "`AfterFileWrite`" + ` | Scan file content after writes | Claude (full); others observe |
| ` + "`BeforePrompt`" + ` | Inspect user prompts for secrets / policy | All five agents |

## Testing Locally

` + "```" + `bash
# Build
go build -o my-hooks .

# Simulate a Claude stop event
echo '{"session_id":"s1","hook_event_name":"Stop","stop_hook_active":false}' \
  | ./my-hooks claude-stop

# Simulate a dangerous shell command (Cursor)
echo '{"command":"rm -rf /","cwd":"/tmp"}' \
  | ./my-hooks cursor-before-shell

# Simulate a prompt with a secret (Claude)
echo '{"session_id":"s1","hook_event_name":"UserPromptSubmit","prompt":"use api_key=abc123"}' \
  | ./my-hooks claude-user-prompt-submit
` + "```" + `
`

const tmplGitignore = `my-hooks
my-hooks.exe
dist/
.env
`

const tmplPolicyJSON = `{
  "_comment": "ILLUSTRATIVE examples only — these substring patterns are trivially bypassed via whitespace, encoding, or indirection. Real policies should use allowlists, command parsing, and input normalization (defense in depth), not naive substring blocklists.",
  "blocked_shell_patterns": [
    "rm -rf /",
    "curl | bash",
    "wget -O- | sh"
  ],
  "allowed_mcp_servers": [],
  "secret_patterns": [
    "api_key=",
    "password=",
    "aws_access_key_id"
  ]
}
`

// copilot-demo is a runnable hook binary for the VS Code Copilot demo.
//
// It demonstrates all four unified hooks firing for Copilot:
//   - BeforeToolCall  — denies dangerous shell commands
//   - WhenAgentIdle   — pushes the agent to keep working unless it's already in a loop
//   - AfterFileWrite  — appends a reminder when *.go files are written
//   - BeforePrompt    — blocks prompts that look like they leak secrets
//
// Build:
//
//	go build -o copilot-demo.exe ./examples/copilot-demo
//
// Install the binary path into your VS Code Copilot config (see README).
package main

import (
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

func main() {
	// NOTE: the banned-command and secret substring matching below is for DEMO
	// purposes only. Substring blocklists are trivially bypassed (whitespace,
	// encoding, indirection) and are not production security — real controls
	// should use allowlists, command parsing, and input normalization.
	agenthooks.BeforeToolCall(func(e agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		if e.IsShell() {
			banned := []string{"rm -rf", "DROP TABLE", "format c:", "shutdown"}
			for _, b := range banned {
				if strings.Contains(strings.ToLower(e.Command), strings.ToLower(b)) {
					return agenthooks.Deny("blocked by cxagenthooks: command contains '" + b + "' (agent=" + string(e.Agent) + ")")
				}
			}
		}
		if e.IsMCP() {
			return agenthooks.AllowWithNote("mcp tool reviewed: " + e.ToolName)
		}
		return agenthooks.Allow()
	})

	agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
		if e.IsLooping() {
			return agenthooks.Resume()
		}
		return agenthooks.Interrupt("Please run tests and confirm before stopping. (cxagenthooks reminder)")
	})

	agenthooks.AfterFileWrite(func(e agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
		if strings.HasSuffix(e.FilePath, ".go") {
			return agenthooks.AnnotateWrite("Reminder from cxagenthooks: run `go vet ./...` before committing.")
		}
		return agenthooks.AcceptWrite()
	})

	agenthooks.BeforePrompt(func(e agenthooks.PromptEvent) agenthooks.PromptVerdict {
		secretPatterns := []string{"API_KEY", "AWS_SECRET", "PRIVATE KEY", "BEGIN RSA"}
		for _, p := range secretPatterns {
			if strings.Contains(e.Text, p) {
				return agenthooks.RejectPrompt("blocked by cxagenthooks: prompt looks like it contains a secret (" + p + ")")
			}
		}
		return agenthooks.AcceptPrompt()
	})

	agenthooks.Dispatch()
}

// Package copilotcli provides types and response helpers for GitHub Copilot CLI
// (and Copilot cloud agent) hooks.
//
// This is a DIFFERENT product from the VS Code Copilot extension modeled by the
// sibling copilot package. The CLI surface differs in three ways that make a
// shared schema impossible:
//
//   - Output is FLAT JSON with no hookSpecificOutput wrapper: preToolUse returns
//     {permissionDecision, permissionDecisionReason, modifiedArgs}; agentStop/
//     subagentStop return top-level {decision, reason}; postToolUse returns
//     {modifiedResult, additionalContext}.
//   - Tool names are lowercase: bash/powershell (shell), create/edit (file write).
//   - Hooks are configured in .github/hooks/*.json (project) or ~/.copilot/hooks/*.json
//     (user) using {"version":1,"hooks":{Event:[{"type":"command","command":...}]}}.
//
// Events are modeled in the VS Code-compatible format (PascalCase event names ->
// snake_case payload fields), which is what `agenthooks install` configures.
//
// See https://docs.github.com/en/copilot/reference/hooks-configuration for the
// official reference.
package copilotcli

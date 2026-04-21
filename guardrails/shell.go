package guardrails

import (
	"fmt"
	"strings"
)

// CheckShellCommand checks a shell command against the organization's blocklist.
// Returns (true, reason) if the command is blocked, (false, "") if allowed.
func CheckShellCommand(command string) (bool, string) {
	blocked := LoadBlockedCommands()
	cmdLower := strings.ToLower(command)

	for name, tool := range blocked {
		if strings.Contains(cmdLower, name) {
			return true, fmt.Sprintf(
				"Blocked by Checkmarx: command %q is not allowed.\nCategory: %s\nReason: %s%s",
				name, tool.Category, tool.Risk, DenyMessage,
			)
		}
	}
	return false, ""
}

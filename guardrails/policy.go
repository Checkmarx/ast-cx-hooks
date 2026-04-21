package guardrails

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DenyMessage is the firm instruction appended to every denial. It tells the
// agent to stop — no retries, no workarounds, no alternative approaches.
const DenyMessage = "\n\n" +
	"IMPORTANT: This action was blocked by your organization's security policy. " +
	"Do NOT attempt alternative commands, workarounds, or equivalent approaches to achieve the same result. " +
	"Any alternative method to accomplish this blocked action may be equally harmful to the system and the developer. " +
	"Instead, inform the user that this operation is restricted by organizational policy and suggest they contact their administrator if they need this action performed."

// HooksPolicy is the root shape of ~/.checkmarx/policyhooks1.json
type HooksPolicy struct {
	DefaultPolicy DefaultPolicy `json:"default_policy"`
}

// DefaultPolicy holds all policy sections.
type DefaultPolicy struct {
	BlocklistTools struct {
		Enabled bool          `json:"enabled"`
		Tools   []BlockedTool `json:"tools"`
	} `json:"blocklist_tools"`
	RestrictedDirectories PathPolicy    `json:"restricted_directories"`
	RestrictedFiles       PathPolicy    `json:"restricted_files"`
	ContextPolicy         ContextPolicy `json:"context_policy"`
}

// ContextPolicy controls what data may enter the AI's context window.
type ContextPolicy struct {
	Enabled         bool            `json:"enabled"`
	ContentScanning ContentScanning `json:"content_scanning"`
}

// ContentScanning holds the scanning configuration.
type ContentScanning struct {
	Enabled  bool                 `json:"enabled"`
	Patterns []ContentScanPattern `json:"patterns"`
}

// ContentScanPattern is a single regex rule that blocks sensitive content in prompts.
type ContentScanPattern struct {
	ID          string `json:"id"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
}

// PathPolicy holds OS-specific path lists for restricted files/directories.
type PathPolicy struct {
	Enabled bool     `json:"enabled"`
	Linux   []string `json:"linux"`
	Windows []string `json:"windows"`
	Mac     []string `json:"mac"`
}

// BlockedTool is a single entry in the shell command blocklist.
type BlockedTool struct {
	Name     string   `json:"name"`
	OS       []string `json:"os"`
	Category string   `json:"category"`
	Risk     string   `json:"risk"`
}

// ShellPolicyPath returns the path to the policy file: ~/.checkmarx/policyhooks1.json
func ShellPolicyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".checkmarx", "policyhooks1.json")
}

// LoadPolicy reads and parses ~/.checkmarx/policyhooks1.json.
// Returns nil on any error (fail-open: a missing or malformed policy should never block the developer).
func LoadPolicy() *HooksPolicy {
	data, err := os.ReadFile(ShellPolicyPath())
	if err != nil {
		return nil
	}
	var p HooksPolicy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil
	}
	return &p
}

// GetOSPaths returns the path list for the current OS from a PathPolicy entry.
func GetOSPaths(pp PathPolicy) []string {
	if !pp.Enabled {
		return nil
	}
	switch runtime.GOOS {
	case "linux":
		return pp.Linux
	case "darwin":
		return pp.Mac
	case "windows":
		return pp.Windows
	default:
		return nil
	}
}

// MatchesOS returns true when any of the tool's OS labels match the current OS.
func MatchesOS(toolOS []string, currentOS string) bool {
	for _, o := range toolOS {
		mapped := o
		if o == "mac" {
			mapped = "darwin"
		}
		if mapped == currentOS {
			return true
		}
	}
	return false
}

// LoadBlockedCommands reads the policy file and returns all command names
// (lowercased) that are blocked on the current OS, together with their metadata.
func LoadBlockedCommands() map[string]BlockedTool {
	blocked := map[string]BlockedTool{}
	policy := LoadPolicy()
	if policy == nil {
		return blocked // fail-open
	}
	if !policy.DefaultPolicy.BlocklistTools.Enabled {
		return blocked
	}
	for _, t := range policy.DefaultPolicy.BlocklistTools.Tools {
		if !MatchesOS(t.OS, runtime.GOOS) {
			continue
		}
		blocked[strings.ToLower(t.Name)] = t
	}
	return blocked
}

// LoadRestrictedPaths returns the OS-specific restricted file and directory
// lists from the policy file.
func LoadRestrictedPaths() (files []string, dirs []string) {
	policy := LoadPolicy()
	if policy == nil {
		return nil, nil
	}
	return GetOSPaths(policy.DefaultPolicy.RestrictedFiles),
		GetOSPaths(policy.DefaultPolicy.RestrictedDirectories)
}

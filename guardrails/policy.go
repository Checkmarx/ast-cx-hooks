package guardrails

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
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
	Tools         ToolsPolicy   `json:"tools"`
}

// DefaultPolicy holds all policy sections that apply at the global scope.
type DefaultPolicy struct {
	BlacklistTools struct {
		Enabled bool              `json:"enabled"`
		Tools   []BlacklistedTool `json:"tools"`
	} `json:"blacklist_tools"`
	RestrictedDirectories PathPolicy       `json:"restricted_directories"`
	RestrictedFiles       PathPolicy       `json:"restricted_files"`
	AllowedDirectories    PathPolicy       `json:"allowed_directories"`
	AllowedFiles          PathPolicy       `json:"allowed_files"`
	ContextPolicy         ContextPolicy    `json:"context_policy"`
	BlastRadiusLimit      BlastRadiusLimit `json:"blast_radius_limit"`
}

// ContextPolicy controls what data may enter the AI's context window.
type ContextPolicy struct {
	Enabled           bool              `json:"enabled"`
	FilesLimits       FilesLimits       `json:"files_limits"`
	ContentScanning   ContentScanning   `json:"content_scanning"`
	BlockedExtensions BlockedExtensions `json:"blocked_extensions"`
}

// FilesLimits restricts how many (and how large) files may be referenced in an AI context.
type FilesLimits struct {
	Enabled       bool `json:"enabled"`
	MaxFileCount  int  `json:"max_file_count"`
	MaxFileSizeKB int  `json:"max_file_size_kb"`
}

// BlockedExtensions lists file extensions that must never enter the AI context.
type BlockedExtensions struct {
	Enabled    bool     `json:"enabled"`
	Extensions []string `json:"extensions"`
}

// BlastRadiusLimit caps how many files the AI may write during a single session.
type BlastRadiusLimit struct {
	Enabled   bool `json:"enabled"`
	Threshold int  `json:"threshold"`
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

// PathPolicy holds OS-specific path lists for restricted or allowed files/directories.
type PathPolicy struct {
	Enabled bool     `json:"enabled"`
	Linux   []string `json:"linux"`
	Windows []string `json:"windows"`
	Mac     []string `json:"mac"`
}

// BlacklistedTool is a single entry in the shell command blacklist.
type BlacklistedTool struct {
	Name     string   `json:"name"`
	OS       []string `json:"os"`
	Category string   `json:"category"`
	Risk     string   `json:"risk"`
}

// ToolsPolicy is the root of the per-tool rule section.
type ToolsPolicy struct {
	Enabled         bool       `json:"enabled"`
	DefaultAuditLog bool       `json:"default_audit_log"`
	Rules           []ToolRule `json:"rules"`
}

// ToolRule defines restrictions and permissions for a specific shell tool.
// Enabled uses *bool so a missing field (nil) is treated as active, preserving
// backward-compatibility with older policy files that don't set the flag.
type ToolRule struct {
	Enabled               *bool         `json:"enabled,omitempty"`
	ID                    string        `json:"id"`
	Tool                  []string      `json:"tool"`
	OS                    []string      `json:"os"`
	ArgsInclude           []string      `json:"args_include"`
	ArgsExclude           []string      `json:"args_exclude"`
	RestrictedDirectories PathPolicy    `json:"restricted_directories"`
	RestrictedFiles       PathPolicy    `json:"restricted_files"`
	AllowedDirectories    PathPolicy    `json:"allowed_directories"`
	AllowedFiles          PathPolicy    `json:"allowed_files"`
	MergeStrategy         MergeStrategy `json:"merge_strategy"`
	AuditLog              bool          `json:"audit_log"`
}

// MergeStrategy controls how a tool rule's path lists are combined with the
// global default_policy values. Applied independently per field.
//
//	merge    = global ∪ rule list
//	override = rule list only (replaces global)
//	default  = global list only (rule's values ignored)
type MergeStrategy struct {
	RestrictedDirectories string `json:"restricted_directories"`
	RestrictedFiles       string `json:"restricted_files"`
	AllowedDirectories    string `json:"allowed_directories"`
	AllowedFiles          string `json:"allowed_files"`
}

// blastRadiusCount tracks how many files have been written during this session.
// Kept as a package-level atomic so concurrent AfterFileWrite handlers are safe.
var blastRadiusCount int32

// ResetBlastRadiusCount resets the session-level file write counter. Exposed for tests.
func ResetBlastRadiusCount() {
	atomic.StoreInt32(&blastRadiusCount, 0)
}

// CheckAndIncrementBlastRadius increments the file-write counter and returns
// blocked=true with a reason if the configured threshold has been exceeded.
func CheckAndIncrementBlastRadius() (blocked bool, reason string) {
	limit := LoadBlastRadiusLimit()
	if limit == nil || !limit.Enabled || limit.Threshold <= 0 {
		return false, ""
	}
	count := int(atomic.AddInt32(&blastRadiusCount, 1))
	if count > limit.Threshold {
		return true, fmt.Sprintf(
			"Blocked by Checkmarx: blast radius limit exceeded. "+
				"This session has written %d files, exceeding the policy threshold of %d.%s",
			count, limit.Threshold, DenyMessage,
		)
	}
	return false, ""
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

// LoadBlacklistedCommands reads the policy file and returns all command names
// (lowercased) that are blacklisted on the current OS, together with their metadata.
func LoadBlacklistedCommands() map[string]BlacklistedTool {
	blacklisted := map[string]BlacklistedTool{}
	policy := LoadPolicy()
	if policy == nil {
		return blacklisted // fail-open
	}
	if !policy.DefaultPolicy.BlacklistTools.Enabled {
		return blacklisted
	}
	for _, t := range policy.DefaultPolicy.BlacklistTools.Tools {
		if !MatchesOS(t.OS, runtime.GOOS) {
			continue
		}
		blacklisted[strings.ToLower(t.Name)] = t
	}
	return blacklisted
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

// LoadAllowedPaths returns the OS-specific allowed file and directory
// lists from the policy file.
func LoadAllowedPaths() (files []string, dirs []string) {
	policy := LoadPolicy()
	if policy == nil {
		return nil, nil
	}
	return GetOSPaths(policy.DefaultPolicy.AllowedFiles),
		GetOSPaths(policy.DefaultPolicy.AllowedDirectories)
}

// LoadBlastRadiusLimit returns the blast-radius limit config, or nil if disabled / absent.
func LoadBlastRadiusLimit() *BlastRadiusLimit {
	policy := LoadPolicy()
	if policy == nil {
		return nil
	}
	limit := policy.DefaultPolicy.BlastRadiusLimit
	if !limit.Enabled {
		return nil
	}
	return &limit
}

// LoadBlockedExtensions returns the list of file extensions blocked from AI context.
// Returns nil when the feature is disabled or the policy is absent.
func LoadBlockedExtensions() []string {
	policy := LoadPolicy()
	if policy == nil {
		return nil
	}
	cp := policy.DefaultPolicy.ContextPolicy
	if !cp.Enabled || !cp.BlockedExtensions.Enabled {
		return nil
	}
	return cp.BlockedExtensions.Extensions
}

// LoadFilesLimits returns the files-limits config, or nil if disabled / absent.
func LoadFilesLimits() *FilesLimits {
	policy := LoadPolicy()
	if policy == nil {
		return nil
	}
	cp := policy.DefaultPolicy.ContextPolicy
	if !cp.Enabled || !cp.FilesLimits.Enabled {
		return nil
	}
	fl := cp.FilesLimits
	return &fl
}

// FindMatchingToolRule returns the first tool rule whose tool list contains the
// base command name and whose OS list matches the current OS. Returns nil if no
// rule matches, the tools section is disabled, or the rule is explicitly disabled.
func FindMatchingToolRule(command string) *ToolRule {
	policy := LoadPolicy()
	if policy == nil || !policy.Tools.Enabled {
		return nil
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return nil
	}
	base := strings.ToLower(fields[0])
	for i := range policy.Tools.Rules {
		rule := &policy.Tools.Rules[i]
		// Explicit `enabled: false` disables a rule; nil (absent) keeps it active.
		if rule.Enabled != nil && !*rule.Enabled {
			continue
		}
		if len(rule.OS) > 0 && !MatchesOS(rule.OS, runtime.GOOS) {
			continue
		}
		for _, name := range rule.Tool {
			if strings.ToLower(name) == base {
				return rule
			}
		}
	}
	return nil
}

// ResolveAllowedPaths combines globalPaths and rulePaths according to strategy.
// Valid strategies: "merge", "override", "default" (anything else acts as "default").
func ResolveAllowedPaths(globalPaths, rulePaths []string, strategy string) []string {
	switch strategy {
	case "merge":
		seen := map[string]struct{}{}
		result := make([]string, 0, len(globalPaths)+len(rulePaths))
		for _, p := range append(globalPaths, rulePaths...) {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				result = append(result, p)
			}
		}
		return result
	case "override":
		return rulePaths
	default: // "default" or anything unrecognised
		return globalPaths
	}
}

// ResolveRestrictedPaths combines global and tool-level restricted paths per strategy.
// Semantics are identical to ResolveAllowedPaths; the two names exist for readability
// at call sites that work with different path categories.
func ResolveRestrictedPaths(globalPaths, rulePaths []string, strategy string) []string {
	return ResolveAllowedPaths(globalPaths, rulePaths, strategy)
}

// NormalizeWorkspaceRoot canonicalises a workspace root so it can be compared
// against policy path entries. Cursor reports Windows roots as "/c:/foo/bar";
// strip the leading slash before a drive letter so PathUnderAny's prefix match
// lines up with policy entries like "C:\\foo\\bar\\".
func NormalizeWorkspaceRoot(root string) string {
	r := filepath.ToSlash(root)
	if len(r) >= 3 && r[0] == '/' && isASCIILetter(r[1]) && r[2] == ':' {
		r = r[1:]
	}
	return r
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

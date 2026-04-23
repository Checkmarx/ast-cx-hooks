package guardrails

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	scanner "github.com/checkmarx/2ms/v3/pkg"
)

// defaultReferencedFileMaxBytes caps per-file reads when the policy does not
// configure files_limits.max_file_size_kb. 1 MB is well above any realistic
// config/source file while preventing accidental reads of large binaries.
const defaultReferencedFileMaxBytes = 1 << 20

// filePathRegexps extracts file/directory paths from free-form text such as user prompts.
// Patterns are tried in order; each must produce the path in the last capture group (or m[0]).
var filePathRegexps = []*regexp.Regexp{
	// @-mention (Cursor/IDE file reference): @.env  @src/config.js  @/absolute/path
	regexp.MustCompile(`@([^\s"'` + "`" + `<>|*?,;:]+)`),
	// Unix absolute: /path/to/file
	regexp.MustCompile(`(?:^|[\s"'` + "`" + `])(/[^\s"'` + "`" + `<>|*?]+)`),
	// Windows absolute: C:\path or C:/path
	regexp.MustCompile(`[A-Za-z]:[\\\/][^\s"'` + "`" + `<>|*?]+`),
	// Explicit relative: ./foo or ../foo
	regexp.MustCompile(`(?:^|[\s"'` + "`" + `])(\.{1,2}/[^\s"'` + "`" + `<>|*?]+)`),
	// Bare dotfile or named file with extension, preceded by space/@/quote/backtick
	// Matches: .env  .env.local  credentials.json  secrets.yaml  id_rsa
	regexp.MustCompile(`(?:^|[\s"'` + "`" + `@])(\.[a-zA-Z0-9][a-zA-Z0-9_.-]*|[a-zA-Z0-9_-]+\.[a-zA-Z0-9][a-zA-Z0-9_.-]*)(?:[\s"'` + "`" + `,;:!?]|$)`),
}

// extractFilePaths returns all file/directory paths found in text, deduplicated.
func extractFilePaths(text string) []string {
	seen := map[string]struct{}{}
	var paths []string
	for _, re := range filePathRegexps {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			p := strings.TrimSpace(m[len(m)-1])
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// severityFromValidation maps 2ms validation status to a severity label.
func severityFromValidation(status string) string {
	switch status {
	case "Valid":
		return "Critical"
	case "Invalid":
		return "Medium"
	default: // "Unknown" or anything else
		return "High"
	}
}

// ScanForSecrets runs the 2ms secret scanner on arbitrary text (e.g. a prompt).
// Returns a human-readable rejection reason, or "" when the text is clean.
func ScanForSecrets(text string) string {
	content := text
	report, err := scanner.NewScanner().Scan(
		[]scanner.ScanItem{{Content: &content, Source: "prompt"}},
		scanner.ScanConfig{WithValidation: true},
	)
	if err != nil {
		return "" // fail-open: scanner error should not block the developer
	}

	var findings []string
	for _, group := range report.Results {
		for _, secret := range group {
			severity := severityFromValidation(string(secret.ValidationStatus))
			findings = append(findings, fmt.Sprintf("  - %s (severity: %s)", secret.RuleID, severity))
		}
	}
	if len(findings) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"Blocked by Checkmarx: prompt contains %d secret(s):\n%s\nRemove the secrets and try again.",
		len(findings), strings.Join(findings, "\n"),
	)
}

// resolveReferencedFile returns an absolute path to a readable regular file for
// the given prompt-referenced path, or "" if the path cannot be resolved to an
// existing file within the known workspace roots.
//
// Absolute paths are used as-is; relative paths are tried against each
// workspace root in order. The first match that resolves to a regular file
// wins. Directories, symlinks to directories, and missing entries return "".
func resolveReferencedFile(p string, workspaceRoots []string) string {
	tryStat := func(path string) string {
		info, err := os.Stat(path)
		if err != nil {
			return ""
		}
		if !info.Mode().IsRegular() {
			return ""
		}
		return path
	}

	if filepath.IsAbs(p) {
		return tryStat(p)
	}
	// Cursor sometimes reports Windows roots as "/c:/foo"; normalise before joining.
	for _, root := range workspaceRoots {
		normalized := NormalizeWorkspaceRoot(root)
		if normalized == "" {
			continue
		}
		if resolved := tryStat(filepath.Join(normalized, p)); resolved != "" {
			return resolved
		}
	}
	return ""
}

// ScanReferencedFiles resolves file paths mentioned in text against the given
// workspace roots, reads each one, and runs the 2ms secret scanner over its
// contents. Returns a human-readable rejection reason that lists findings per
// file, or "" when no referenced file contains secrets.
//
// Missing files, directories, and files that exceed the configured size cap
// are silently skipped — this is a best-effort guardrail, not a filesystem
// audit, and must not block the developer on unrelated I/O errors.
func ScanReferencedFiles(text string, workspaceRoots []string) string {
	paths := extractFilePaths(text)
	if len(paths) == 0 {
		return ""
	}

	maxBytes := int64(defaultReferencedFileMaxBytes)
	if limits := LoadFilesLimits(); limits != nil && limits.MaxFileSizeKB > 0 {
		maxBytes = int64(limits.MaxFileSizeKB) * 1024
	}

	seen := map[string]struct{}{}
	var perFile []string
	sc := scanner.NewScanner()

	for _, p := range paths {
		resolved := resolveReferencedFile(p, workspaceRoots)
		if resolved == "" {
			continue
		}
		if _, dup := seen[resolved]; dup {
			continue
		}
		seen[resolved] = struct{}{}

		info, err := os.Stat(resolved)
		if err != nil || info.Size() > maxBytes {
			continue
		}

		data, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}
		content := string(data)

		report, err := sc.Scan(
			[]scanner.ScanItem{{Content: &content, Source: resolved}},
			scanner.ScanConfig{WithValidation: true},
		)
		if err != nil {
			continue // fail-open per scanner
		}

		var findings []string
		for _, group := range report.Results {
			for _, secret := range group {
				severity := severityFromValidation(string(secret.ValidationStatus))
				findings = append(findings, fmt.Sprintf("    - %s (severity: %s)", secret.RuleID, severity))
			}
		}
		if len(findings) == 0 {
			continue
		}
		perFile = append(perFile,
			fmt.Sprintf("  %s (%d secret(s)):\n%s", p, len(findings), strings.Join(findings, "\n")))
	}

	if len(perFile) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"Blocked by Checkmarx: referenced file(s) contain secrets:\n%s\nRemove the secrets from these files or remove the references from your prompt.%s",
		strings.Join(perFile, "\n"), DenyMessage,
	)
}

// ScanForPolicyPatterns runs the custom regex patterns defined in the policy's
// context_policy.content_scanning section against the prompt text.
// Returns a human-readable rejection reason, or "" when the text is clean.
func ScanForPolicyPatterns(text string) string {
	policy := LoadPolicy()
	if policy == nil {
		return ""
	}
	cp := policy.DefaultPolicy.ContextPolicy
	if !cp.Enabled || !cp.ContentScanning.Enabled {
		return ""
	}

	var findings []string
	for _, p := range cp.ContentScanning.Patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			continue // skip malformed patterns — fail-open
		}
		if re.MatchString(text) {
			findings = append(findings, fmt.Sprintf("  - %s: %s", p.ID, p.Description))
		}
	}
	if len(findings) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"Blocked by Checkmarx: prompt contains sensitive content detected by policy:\n%s\nRemove the sensitive content and try again.%s",
		strings.Join(findings, "\n"), DenyMessage,
	)
}

// CheckPromptPaths checks file/directory paths mentioned in a prompt against
// the organization's restricted_files and restricted_directories policy.
//
// Precedence: restricted always wins over allowed. A path that matches both a
// restricted and an allowed list is still blocked.
//
// Returns (true, reason) if blocked, (false, "") if allowed.
func CheckPromptPaths(text string) (bool, string) {
	restrictedFiles, restrictedDirs := LoadRestrictedPaths()
	if len(restrictedFiles) == 0 && len(restrictedDirs) == 0 {
		return false, ""
	}

	files := extractFilePaths(text)
	seen := map[string]struct{}{}
	var violations []string

	for _, file := range files {
		// restricted_files: literal, basename, suffix, or doublestar glob.
		for _, rf := range restrictedFiles {
			if matchFilePattern(rf, file) {
				if _, ok := seen[file]; !ok {
					seen[file] = struct{}{}
					violations = append(violations, fmt.Sprintf("  - %s (restricted file)", file))
				}
				break
			}
		}

		// restricted_directories: containment match, with glob support.
		if _, already := seen[file]; already {
			continue
		}
		for _, rd := range restrictedDirs {
			if matchDirContains(rd, file) {
				seen[file] = struct{}{}
				violations = append(violations, fmt.Sprintf("  - %s (restricted directory)", file))
				break
			}
		}
	}

	if len(violations) == 0 {
		return false, ""
	}
	return true, fmt.Sprintf(
		"Blocked by Checkmarx: the following files or folders are restricted by policy:\n%s\nContact your administrator if you need access to these resources.%s",
		strings.Join(violations, "\n"), DenyMessage,
	)
}

// CheckWorkspaceRoots rejects a prompt whose workspace is within a restricted directory.
// Policy entries are interpreted per-OS via LoadRestrictedPaths; the prefix match
// makes a workspace at C:\foo\bar illegal when C:\foo\ is restricted.
// Returns (true, reason) if any root violates policy, (false, "") otherwise.
func CheckWorkspaceRoots(roots []string) (bool, string) {
	if len(roots) == 0 {
		return false, ""
	}
	_, restrictedDirs := LoadRestrictedPaths()
	if len(restrictedDirs) == 0 {
		return false, ""
	}
	for _, root := range roots {
		normalized := NormalizeWorkspaceRoot(root)
		if PathUnderAny(normalized, restrictedDirs) {
			return true, fmt.Sprintf(
				"Blocked by Checkmarx: workspace %q is restricted by policy.%s",
				root, DenyMessage,
			)
		}
	}
	return false, ""
}

// CheckBlockedExtensions rejects prompts that reference files with a blocked extension
// (e.g. .env, .pem, .key). Returns a rejection reason, or "" when the prompt is clean.
func CheckBlockedExtensions(text string) string {
	extensions := LoadBlockedExtensions()
	if len(extensions) == 0 {
		return ""
	}
	extSet := make(map[string]struct{}, len(extensions))
	for _, e := range extensions {
		extSet[strings.ToLower(e)] = struct{}{}
	}

	seen := map[string]struct{}{}
	var hits []string
	for _, p := range extractFilePaths(text) {
		ext := strings.ToLower(filepath.Ext(p))
		if ext == "" {
			continue
		}
		if _, ok := extSet[ext]; !ok {
			continue
		}
		if _, already := seen[p]; already {
			continue
		}
		seen[p] = struct{}{}
		hits = append(hits, fmt.Sprintf("  - %s (extension %s)", p, ext))
	}
	if len(hits) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"Blocked by Checkmarx: prompt references files with blocked extensions:\n%s\nThese file types must not enter the AI context.%s",
		strings.Join(hits, "\n"), DenyMessage,
	)
}

// CheckFilesLimits rejects prompts that reference more files than the policy allows.
// Returns a rejection reason, or "" when the prompt is within the limit.
func CheckFilesLimits(text string) string {
	limits := LoadFilesLimits()
	if limits == nil || limits.MaxFileCount <= 0 {
		return ""
	}
	paths := extractFilePaths(text)
	if len(paths) <= limits.MaxFileCount {
		return ""
	}
	return fmt.Sprintf(
		"Blocked by Checkmarx: prompt references %d files, exceeding the policy limit of %d.%s",
		len(paths), limits.MaxFileCount, DenyMessage,
	)
}

// ScanPrompt runs all prompt guardrails in order:
//  1. 2ms secret scanner      — detects structured secrets (API keys, tokens, PEM blocks)
//  2. Policy content scanner  — detects sensitive content via custom regex patterns
//  3. Path guardrail          — blocks prompts referencing restricted files/directories
//  4. Blocked extensions      — blocks prompts referencing files with blocked extensions
//  5. Files-limits guardrail  — rejects prompts that reference too many files
//
// Returns a human-readable rejection reason, or "" when the text is clean.
func ScanPrompt(text string) string {
	if reason := ScanForSecrets(text); reason != "" {
		return reason
	}
	if reason := ScanForPolicyPatterns(text); reason != "" {
		return reason
	}
	if blocked, reason := CheckPromptPaths(text); blocked {
		return reason
	}
	if reason := CheckBlockedExtensions(text); reason != "" {
		return reason
	}
	if reason := CheckFilesLimits(text); reason != "" {
		return reason
	}
	return ""
}

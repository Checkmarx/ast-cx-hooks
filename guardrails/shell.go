package guardrails

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// CheckShellCommand checks a shell command against the blacklist and per-tool rules.
//
// Returns:
//   - blocked=true, needsConfirm=false → deny the command
//   - blocked=true, needsConfirm=true  → ask the user for confirmation
//   - blocked=false                    → permit the command
//
// Enforcement order (first hit wins):
//  1. Global blacklist_tools              → hard block
//  2. args_exclude                        → hard block
//  3. Tool-level restricted_dirs / files  → hard block
//  4. Global restricted_dirs / files      → hard block
//  5. args_include whitelist              → ask on first unmatched token
//  6. allowed_directories                 → ask if workDir not in effective list
//  7. allowed_files                       → ask if file arg not in effective list
func CheckShellCommand(command, workDir string) (blocked bool, needsConfirm bool, reason string) {
	// 1. Global blacklist check.
	blacklisted := LoadBlacklistedCommands()
	cmdLower := strings.ToLower(command)
	for name, tool := range blacklisted {
		if strings.Contains(cmdLower, name) {
			return true, false, fmt.Sprintf(
				"Blocked by Checkmarx: command %q is not allowed.\nCategory: %s\nReason: %s%s",
				name, tool.Category, tool.Risk, DenyMessage,
			)
		}
	}

	// 2. Per-tool rule enforcement.
	rule := FindMatchingToolRule(command)
	if rule == nil {
		// No tool rule matched — still enforce global restricted paths.
		return checkGlobalRestrictedPaths(command, workDir)
	}

	// 2a. args_exclude — hard deny, takes precedence over everything else.
	for _, excluded := range rule.ArgsExclude {
		if strings.Contains(cmdLower, strings.ToLower(excluded)) {
			return true, false, fmt.Sprintf(
				"Blocked by Checkmarx: argument %q is not permitted for this tool.\nContact your administrator if you need this operation.%s",
				excluded, DenyMessage,
			)
		}
	}

	// 2b. Tool-level restricted_directories / restricted_files (merged per strategy).
	if blocked, needsConfirm, reason := checkToolRestrictedPaths(command, workDir, rule); blocked {
		return blocked, needsConfirm, reason
	}

	// 2c. args_include whitelist — any token not matched by an entry (exact or glob)
	// triggers a confirmation request regardless of rule.Action. This is intentionally
	// "ask" rather than "block" because an unknown arg is not necessarily dangerous;
	// it is simply unknown to the policy author.
	if len(rule.ArgsInclude) > 0 {
		tokens := strings.Fields(command)
		if len(tokens) > 1 { // skip the command name itself (tokens[0])
			for _, tok := range tokens[1:] {
				if !argMatchesAny(tok, rule.ArgsInclude) {
					msg := fmt.Sprintf(
						"Argument %q is not in the approved list for this tool.%s",
						tok, DenyMessage,
					)
					return true, true, msg // always ask on first unmatched token
				}
			}
		}
	}

	// 2d. Allowed-directories check — if a list is defined, workDir must be inside it.
	// Unknown (not in list) → ask.
	if workDir != "" {
		_, globalDirs := LoadAllowedPaths()
		effectiveDirs := ResolveAllowedPaths(globalDirs, GetOSPaths(rule.AllowedDirectories), rule.MergeStrategy.AllowedDirectories)
		if len(effectiveDirs) > 0 && !pathUnderAny(workDir, effectiveDirs) {
			return true, true, fmt.Sprintf(
				"Working directory %q is not in the allowed list for this tool.%s",
				workDir, DenyMessage,
			)
		}
	}

	// 2e. Allowed-files check — file-like tokens in the command must be in the effective list.
	// Unknown (not in list) → ask.
	globalFiles, _ := LoadAllowedPaths()
	effectiveFiles := ResolveAllowedPaths(globalFiles, GetOSPaths(rule.AllowedFiles), rule.MergeStrategy.AllowedFiles)
	if len(effectiveFiles) > 0 {
		fileSet := make(map[string]struct{}, len(effectiveFiles))
		for _, f := range effectiveFiles {
			fileSet[strings.ToLower(filepath.Base(f))] = struct{}{}
		}
		tokens := strings.Fields(command)
		if len(tokens) > 1 {
			for _, token := range tokens[1:] {
				// Only check tokens that look like file names (contain a path separator or a dot).
				if strings.ContainsAny(token, "./\\") || strings.Contains(token, ".xml") || strings.Contains(token, ".json") {
					base := strings.ToLower(filepath.Base(token))
					if _, ok := fileSet[base]; !ok {
						return true, true, fmt.Sprintf(
							"File %q is not in the allowed list for this tool.%s",
							token, DenyMessage,
						)
					}
				}
			}
		}
	}

	return false, false, ""
}

// checkToolRestrictedPaths enforces restricted_directories and restricted_files for a
// matched tool rule, merging tool-level paths with the global lists per the rule's
// merge_strategy. Violations are always a hard block (needsConfirm=false).
func checkToolRestrictedPaths(command, workDir string, rule *ToolRule) (bool, bool, string) {
	globalFiles, globalDirs := LoadRestrictedPaths()

	// Restricted directories: workDir must not fall under any effective restricted dir.
	effectiveDirs := ResolveRestrictedPaths(globalDirs, GetOSPaths(rule.RestrictedDirectories), rule.MergeStrategy.RestrictedDirectories)
	if workDir != "" && len(effectiveDirs) > 0 && pathUnderAny(workDir, effectiveDirs) {
		return true, false, fmt.Sprintf(
			"Blocked by Checkmarx: working directory %q is restricted by policy and not permitted for this tool.%s",
			workDir, DenyMessage,
		)
	}

	// Restricted files: no file arg may match an effective restricted file.
	effectiveFiles := ResolveRestrictedPaths(globalFiles, GetOSPaths(rule.RestrictedFiles), rule.MergeStrategy.RestrictedFiles)
	if len(effectiveFiles) > 0 {
		if hit := findRestrictedFileInCommand(command, effectiveFiles); hit != "" {
			return true, false, fmt.Sprintf(
				"Blocked by Checkmarx: file %q is restricted by policy and not permitted for this tool.%s",
				hit, DenyMessage,
			)
		}
	}
	return false, false, ""
}

// checkGlobalRestrictedPaths enforces the global restricted_directories and
// restricted_files when no tool rule matches the command. Always a hard block.
func checkGlobalRestrictedPaths(command, workDir string) (bool, bool, string) {
	globalFiles, globalDirs := LoadRestrictedPaths()
	if len(globalFiles) == 0 && len(globalDirs) == 0 {
		return false, false, ""
	}

	if workDir != "" && len(globalDirs) > 0 && pathUnderAny(workDir, globalDirs) {
		return true, false, fmt.Sprintf(
			"Blocked by Checkmarx: working directory %q is restricted by policy.%s",
			workDir, DenyMessage,
		)
	}
	if hit := findRestrictedFileInCommand(command, globalFiles); hit != "" {
		return true, false, fmt.Sprintf(
			"Blocked by Checkmarx: file %q is restricted by policy.%s",
			hit, DenyMessage,
		)
	}
	return false, false, ""
}

// findRestrictedFileInCommand returns the first token in the command (skipping
// the command name itself) that matches a restricted file by exact path, basename,
// or path suffix. Returns "" when no token matches.
func findRestrictedFileInCommand(command string, restrictedFiles []string) string {
	tokens := strings.Fields(command)
	if len(tokens) <= 1 {
		return ""
	}
	restrictedSet := make(map[string]struct{}, len(restrictedFiles))
	for _, f := range restrictedFiles {
		restrictedSet[strings.ToLower(filepath.ToSlash(f))] = struct{}{}
	}
	for _, token := range tokens[1:] {
		if !strings.ContainsAny(token, "./\\") {
			continue
		}
		tokLower := strings.ToLower(filepath.ToSlash(token))
		tokBase := strings.ToLower(filepath.Base(token))
		for _, rf := range restrictedFiles {
			rfLower := strings.ToLower(filepath.ToSlash(rf))
			if tokLower == rfLower ||
				tokBase == rfLower ||
				strings.HasSuffix(tokLower, "/"+rfLower) {
				return token
			}
		}
	}
	return ""
}

// argMatchesAny returns true when arg matches at least one entry in patterns
// using case-insensitive exact match or path.Match glob syntax (e.g. "-D*", "--*").
func argMatchesAny(arg string, patterns []string) bool {
	argLower := strings.ToLower(arg)
	for _, p := range patterns {
		pLower := strings.ToLower(p)
		if argLower == pLower {
			return true
		}
		if matched, _ := path.Match(pLower, argLower); matched {
			return true
		}
	}
	return false
}

// pathUnderAny returns true when path falls within at least one of the candidate directories.
func pathUnderAny(path string, dirs []string) bool {
	pathSlash := strings.ToLower(filepath.ToSlash(path))
	for _, d := range dirs {
		dSlash := strings.ToLower(strings.TrimSuffix(filepath.ToSlash(d), "/"))
		if pathSlash == dSlash || strings.HasPrefix(pathSlash, dSlash+"/") {
			return true
		}
	}
	return false
}

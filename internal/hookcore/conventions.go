package hookcore

import (
	"encoding/json"
	"strings"
)

// ToolConvention captures one agent's tool-naming scheme: how to recognize its
// shell and MCP tools, which tools write files, and where in tool_input the file
// path and edit text live. It replaces five near-duplicate helper families with a
// single deep type plus per-agent data instances, so the tool-naming knowledge for
// every supported agent lives in one place.
type ToolConvention struct {
	ShellTools   []string // tool names whose tool_input.command is a shell command
	MCPPrefix    string   // tool-name prefix marking an MCP tool ("" disables MCP matching)
	WriteTools   []string // tool names that create or modify files
	FilePathKeys []string // tool_input keys to try, in order, for the edited file path
	// Diff extracts before/after edit text; nil means this agent reports no diffs.
	Diff func(toolName string, input json.RawMessage) []FileDiff
	// FilePathFromDiff derives a file path from the raw patch/diff text when no
	// FilePathKeys entry matches (e.g. Codex's apply_patch, which has no
	// separate file_path key). nil means this agent has no such fallback.
	FilePathFromDiff func(toolName string, input json.RawMessage) string
}

// Kind classifies a tool name into shell / MCP / builtin, returning the shell
// command (from tool_input.command) when the tool is a shell tool.
func (c ToolConvention) Kind(toolName string, input json.RawMessage) (ToolKind, string) {
	if c.MCPPrefix != "" && strings.HasPrefix(toolName, c.MCPPrefix) {
		return ToolKindMCP, ""
	}
	for _, s := range c.ShellTools {
		if toolName == s {
			var v struct {
				Command string `json:"command"`
			}
			json.Unmarshal(input, &v) //nolint:errcheck // absent command is fine
			return ToolKindShell, v.Command
		}
	}
	return ToolKindBuiltin, ""
}

// IsWrite reports whether toolName is one of this agent's file-writing tools.
func (c ToolConvention) IsWrite(toolName string) bool {
	for _, w := range c.WriteTools {
		if toolName == w {
			return true
		}
	}
	return false
}

// FilePath returns the edited file path, trying each configured key in order,
// then falling back to FilePathFromDiff (when set) if no key matched.
func (c ToolConvention) FilePath(input json.RawMessage) string {
	var m map[string]json.RawMessage
	if json.Unmarshal(input, &m) != nil {
		return ""
	}
	for _, k := range c.FilePathKeys {
		if raw, ok := m[k]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil && s != "" {
				return s
			}
		}
	}
	if c.FilePathFromDiff != nil {
		return c.FilePathFromDiff("", input)
	}
	return ""
}

// Changes returns the before/after edit text, or nil if this agent reports none.
func (c ToolConvention) Changes(toolName string, input json.RawMessage) []FileDiff {
	if c.Diff == nil {
		return nil
	}
	return c.Diff(toolName, input)
}

// standardDiff handles the Write/Edit/MultiEdit convention (Claude, Copilot):
// Edit/MultiEdit carry old_string/new_string; other write tools carry content.
func standardDiff(toolName string, input json.RawMessage) []FileDiff {
	var v struct {
		Content   string `json:"content"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	if toolName == "Edit" || toolName == "MultiEdit" {
		return []FileDiff{{Before: v.OldString, After: v.NewString}}
	}
	return []FileDiff{{Before: "", After: v.Content}}
}

// droidDiff handles Factory Droid's write tools: Edit carries old_string/new_string,
// ApplyPatch carries a unified patch (patch, else diff) surfaced best-effort as the
// After side, and Create carries content.
func droidDiff(toolName string, input json.RawMessage) []FileDiff {
	var v struct {
		Content   string `json:"content"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
		Patch     string `json:"patch"`
		Diff      string `json:"diff"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	switch toolName {
	case "Edit":
		return []FileDiff{{Before: v.OldString, After: v.NewString}}
	case "ApplyPatch":
		patch := v.Patch
		if patch == "" {
			patch = v.Diff
		}
		return []FileDiff{{Before: "", After: patch}}
	default: // Create
		return []FileDiff{{Before: "", After: v.Content}}
	}
}

// cursorDiff handles Cursor Agent write tools. Cursor IDE preToolUse payloads have
// historically used file_path + content; the Cursor CLI / Composer agent tools emit
// path + contents for Write and path + old_string/new_string for StrReplace and
// EditNotebook. All keys are accepted so the same cx hooks gate works in both.
func cursorDiff(toolName string, input json.RawMessage) []FileDiff {
	var v struct {
		Content   string `json:"content"`
		Contents  string `json:"contents"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	switch toolName {
	case "Edit", "StrReplace", "EditNotebook", "MultiEdit":
		return []FileDiff{{Before: v.OldString, After: v.NewString}}
	default:
		content := v.Content
		if content == "" {
			content = v.Contents
		}
		return []FileDiff{{Before: "", After: content}}
	}
}

// cliDiff handles the GitHub Copilot CLI write tools. Confirmed against a live
// payload: edit carries old_str/new_str and create carries file_text; content
// is kept as a fallback for both.
//
// Pointer fields are used so that a key absent from JSON (nil) is distinguished
// from a key present with an empty-string value (""). A zero-value string check
// would conflate the two and silently skip the primary key even when it is
// legitimately empty — causing ProposedContent to treat the edit as a full-file
// write of only new_str, which breaks delta detection for the ASCA guardrail.
func cliDiff(toolName string, input json.RawMessage) []FileDiff {
	var v struct {
		Content  *string `json:"content"`
		FileText *string `json:"file_text"`
		OldStr   *string `json:"old_str"`
		NewStr   *string `json:"new_str"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	if toolName == "edit" {
		return []FileDiff{{Before: ptrOr(v.OldStr, v.Content), After: ptrOr(v.NewStr, v.Content)}}
	}

	return []FileDiff{{Before: "", After: ptrOr(v.FileText, v.Content)}}
}

// ptrOr returns the dereferenced value of primary when it is non-nil,
// then secondary when non-nil, and "" otherwise.
func ptrOr(primary, secondary *string) string {
	if primary != nil {
		return *primary
	}
	if secondary != nil {
		return *secondary
	}
	return ""
}

// codexPatchCommand extracts the raw patch body from a Codex apply_patch
// tool_input. CONFIRMED against a live payload: the patch text is carried
// under tool_input.command (the same key PreToolUse uses for shell commands),
// not "input" as originally assumed from public examples. "input" is kept as
// a fallback in case some Codex CLI versions use that key instead.
func codexPatchCommand(input json.RawMessage) string {
	var v struct {
		Command string `json:"command"`
		Input   string `json:"input"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	if v.Command != "" {
		return v.Command
	}
	return v.Input
}

// codexFileSection is one file's worth of a Codex apply_patch body: the path
// from its "*** Add/Update/Delete File:" header and the raw lines that follow
// it (up to the next file header or "*** End Patch").
type codexFileSection struct {
	kind string // "add", "update", or "delete"
	path string
	body []string
}

// codexPatchSections splits a Codex apply_patch body into one section per
// touched file, following the official apply_patch grammar (the Lark grammar
// documented in openai/codex's own parser, codex-rs/apply-patch/src/parser.rs):
//
//	start: begin_patch environment_id? hunk+ end_patch
//	begin_patch: "*** Begin Patch" LF
//	environment_id: "*** Environment ID: " filename LF
//	end_patch: "*** End Patch" LF?
//	hunk: add_hunk | delete_hunk | update_hunk
//	add_hunk: "*** Add File: " filename LF add_line+
//	delete_hunk: "*** Delete File: " filename LF
//	update_hunk: "*** Update File: " filename LF change_move? change?
//	change_move: "*** Move to: " filename LF
//	change: (change_context | change_line)+ eof_line?
//	change_context: ("@@" | "@@ " /(.+)/) LF
//	change_line: ("+" | "-" | " ") /(.+)/ LF
//	eof_line: "*** End of File" LF
//
// "*** Begin/Update/Add/End Patch" and the hunk/line format inside an Update
// File section are additionally CONFIRMED against live Codex CLI payloads
// (single- and multi-hunk Update File, Add File with relative and absolute
// Windows paths — see internal/hookcore/conventions_test.go). "*** Delete
// File:", "*** Move to:", and "*** Environment ID:" have not been seen in a
// live payload yet, but their exact marker text and grammar position are
// confirmed against the grammar above, not guessed. A Delete File section
// yields no diff either way (codexDiff's delete case); "*** Environment ID:"
// (a patch-global line before any file section) and "*** Move to:" (consumed
// immediately after its Update File header, before any hunks) are both
// recognized and skipped so they can never be misread as hunk body content.
func codexPatchSections(patch string) []codexFileSection {
	var sections []codexFileSection
	var cur *codexFileSection
	for _, line := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			sections = append(sections, codexFileSection{kind: "add", path: strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: "))})
			cur = &sections[len(sections)-1]
		case strings.HasPrefix(line, "*** Update File: "):
			sections = append(sections, codexFileSection{kind: "update", path: strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: "))})
			cur = &sections[len(sections)-1]
		case strings.HasPrefix(line, "*** Delete File: "):
			sections = append(sections, codexFileSection{kind: "delete", path: strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File: "))})
			cur = &sections[len(sections)-1]
		case line == "*** End Patch":
			// Stops the current section so any trailing content after the
			// marker (e.g. the "" element Split("\n") yields for a patch
			// ending in a newline) isn't appended to the last file's body.
			cur = nil
		case line == "*** Begin Patch",
			strings.HasPrefix(line, "*** Environment ID: "),
			strings.HasPrefix(line, "*** Move to: "),
			line == "*** End of File":
			// Grammar markers that carry no hunk-body content: Begin Patch is
			// the envelope open; Environment ID is a patch-global line before
			// any file section; Move to is consumed right after its Update
			// File header (rename target, not a diff line); End of File
			// terminates a change block to flag it reaches EOF.
		default:
			if cur != nil {
				cur.body = append(cur.body, line)
			}
		}
	}
	return sections
}

// codexHunkDiffs turns an Update File section's body into one FileDiff per
// "@@"-delimited hunk. Context lines (leading space) anchor the hunk in both
// Before and After; "-" lines are removed (Before only); "+" lines are added
// (After only). This mirrors the V4A/unified-diff convention Codex's
// apply_patch uses and lets ProposedContent's substring-replace logic locate
// the hunk in the on-disk file via its context lines, the same way Claude's
// Edit (old_string/new_string) does.
func codexHunkDiffs(body []string) []FileDiff {
	var diffs []FileDiff
	var before, after strings.Builder
	flush := func() {
		if before.Len() > 0 || after.Len() > 0 {
			diffs = append(diffs, FileDiff{Before: before.String(), After: after.String()})
		}
		before.Reset()
		after.Reset()
	}
	for _, line := range body {
		if strings.HasPrefix(line, "@@") {
			flush()
			continue
		}
		if line == "" {
			// A zero-length line is a blank context line lacking even the usual
			// leading " " marker (some patch producers omit it for empty lines,
			// and Split("\n") also yields a trailing "" after the final line) —
			// treat it as shared context so it isn't silently dropped from
			// both Before and After, which would break the anchor match
			// against on-disk content that has a real blank line there.
			before.WriteByte('\n')
			after.WriteByte('\n')
			continue
		}
		switch line[0] {
		case '+':
			after.WriteString(line[1:])
			after.WriteByte('\n')
		case '-':
			before.WriteString(line[1:])
			before.WriteByte('\n')
		case ' ':
			rest := line[1:]
			before.WriteString(rest)
			before.WriteByte('\n')
			after.WriteString(rest)
			after.WriteByte('\n')
		default:
			// Non-standard context line (no leading marker); treat as shared context.
			before.WriteString(line)
			before.WriteByte('\n')
			after.WriteString(line)
			after.WriteByte('\n')
		}
	}
	flush()
	return diffs
}

// codexAddFileContent reconstructs a new file's full content from an Add File
// section's body by stripping the leading "+" every content line carries.
func codexAddFileContent(body []string) string {
	var b strings.Builder
	for _, line := range body {
		b.WriteString(strings.TrimPrefix(line, "+"))
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// codexDiff handles OpenAI Codex CLI's apply_patch tool. apply_patch's
// tool_input.command holds a V4A-style patch body that can touch multiple
// files in one call, each introduced by "*** Add/Update/Delete File: <path>".
// codexDiff surfaces the diffs for the FIRST file section only (matching
// codexPatchFilePath's single-representative-path behavior — FileDiff has no
// per-file grouping, so mixing hunks from multiple files would let
// ProposedContent apply one file's hunks against another file's disk
// content). Add File reconstructs the full new content (Before ""); Update
// File yields one FileDiff per hunk with real before/after text (not raw
// patch syntax) so ASCA/KICS scan actual source instead of diff markers;
// Delete File yields no diff.
func codexDiff(_ string, input json.RawMessage) []FileDiff {
	sections := codexPatchSections(codexPatchCommand(input))
	if len(sections) == 0 {
		return nil
	}
	switch sections[0].kind {
	case "add":
		return []FileDiff{{Before: "", After: codexAddFileContent(sections[0].body)}}
	case "update":
		return codexHunkDiffs(sections[0].body)
	default: // delete
		return nil
	}
}

// codexPatchFilePath extracts the first file path touched by a Codex
// apply_patch call, so downstream file-extension-based guardrails (ASCA/KICS)
// have something to key off even though apply_patch has no dedicated
// file_path field. A single call can touch multiple files; only the first
// section's path is returned (see codexDiff for why diffs are likewise
// scoped to that first file). Returns "" if no file header is present.
func codexPatchFilePath(_ string, input json.RawMessage) string {
	sections := codexPatchSections(codexPatchCommand(input))
	if len(sections) == 0 {
		return ""
	}
	return sections[0].path
}

var (
	// CodexTools is the OpenAI Codex CLI tool-naming convention. The shell tool
	// name (Bash) and MCP prefix (mcp__) are still UNVERIFIED — modeled from
	// https://learn.chatgpt.com/docs/hooks, assumed to mirror Claude's schema.
	// apply_patch has no single file_path key (it's a multi-file V4A patch), so
	// FilePathKeys is empty; FilePath() falls back to codexPatchFilePath. Both
	// codexPatchFilePath and codexDiff are confirmed against live payloads
	// (Add File and multi-hunk Update File, relative and absolute Windows
	// paths) and parse the patch body itself — see codexPatchSections,
	// codexHunkDiffs, codexAddFileContent in this file.
	CodexTools = ToolConvention{
		ShellTools: []string{"Bash"}, MCPPrefix: "mcp__",
		WriteTools:       []string{"apply_patch"},
		FilePathKeys:     nil,
		Diff:             codexDiff,
		FilePathFromDiff: codexPatchFilePath,
	}
)

var (
	// ClaudeTools is the Claude Code tool-naming convention (Bash + mcp__ + Write/Edit/MultiEdit).
	ClaudeTools = ToolConvention{
		ShellTools: []string{"Bash"}, MCPPrefix: "mcp__",
		WriteTools:   []string{"Write", "Edit", "MultiEdit"},
		FilePathKeys: []string{"file_path"}, Diff: standardDiff,
	}
	// DroidTools is the Factory Droid convention (Execute + mcp__ + Create/Edit/ApplyPatch).
	DroidTools = ToolConvention{
		ShellTools: []string{"Execute"}, MCPPrefix: "mcp__",
		WriteTools:   []string{"Create", "Edit", "ApplyPatch"},
		FilePathKeys: []string{"file_path"}, Diff: droidDiff,
	}
	// GeminiTools is the Gemini CLI convention (run_shell_command + single-underscore
	// mcp_ + write_file/replace). Gemini reports no before/after diffs.
	GeminiTools = ToolConvention{
		ShellTools: []string{"run_shell_command"}, MCPPrefix: "mcp_",
		WriteTools:   []string{"write_file", "replace"},
		FilePathKeys: []string{"file_path", "path"}, Diff: nil,
	}
	// CopilotTools is the VS Code Copilot convention (runTerminalCommand + mcp_ +
	// createFile/editFiles, camelCase filePath).
	CopilotTools = ToolConvention{
		ShellTools: []string{"runTerminalCommand"}, MCPPrefix: "mcp_",
		WriteTools:   []string{"createFile", "editFiles"},
		FilePathKeys: []string{"filePath"}, Diff: standardDiff,
	}
	// CopilotCLITools is the GitHub Copilot CLI convention (lowercase bash/powershell,
	// no MCP prefix documented, create/edit). Confirmed against a live payload: the
	// edited file path is in "path" (file_path kept as a fallback).
	CopilotCLITools = ToolConvention{
		ShellTools: []string{"bash", "powershell"}, MCPPrefix: "",
		WriteTools:   []string{"create", "edit"},
		FilePathKeys: []string{"path", "file_path"}, Diff: cliDiff,
	}
	// CursorTools is the Cursor Agent convention (Write/StrReplace/EditNotebook for
	// file mutations). tool_input may use file_path or path, and content or contents.
	CursorTools = ToolConvention{
		WriteTools:   []string{"Write", "Edit", "StrReplace", "EditNotebook"},
		FilePathKeys: []string{"file_path", "path"},
		Diff:         cursorDiff,
	}
)

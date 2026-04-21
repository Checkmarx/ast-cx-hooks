package lifecycle

import (
	"encoding/json"

	"github.com/CheckmarxDev/ast-cx-hooks/claude"
	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
	"github.com/CheckmarxDev/ast-cx-hooks/droid"
	"github.com/CheckmarxDev/ast-cx-hooks/gemini"
	"github.com/CheckmarxDev/ast-cx-hooks/internal/dispatch"
	"github.com/CheckmarxDev/ast-cx-hooks/windsurf"
)

// FileDiff represents a single text replacement in a file write operation.
type FileDiff struct {
	Before string // original content (empty for full-file writes)
	After  string // new content
}

// FileWriteEvent provides a unified view of post-file-edit events.
type FileWriteEvent struct {
	Agent     AgentID
	SessionID string
	FilePath  string
	Changes   []FileDiff
	WorkDir   string
	Raw       any
}

// FileWriteVerdict is the decision returned by an AfterFileWrite handler.
type FileWriteVerdict struct {
	Reject   bool   // true = inject feedback into the agent (Claude/Droid only)
	Feedback string // message sent to the agent when Reject is true
	Footnote string // additional context appended after the edit (Claude/Droid only)
}

// AcceptWrite acknowledges the file write with no feedback.
func AcceptWrite() FileWriteVerdict { return FileWriteVerdict{} }

// RejectWrite injects a rejection message into the agent after the write.
func RejectWrite(reason string) FileWriteVerdict {
	return FileWriteVerdict{Reject: true, Feedback: reason}
}

// AnnotateWrite appends a note the agent will see after the write.
func AnnotateWrite(note string) FileWriteVerdict { return FileWriteVerdict{Footnote: note} }

// FileWriteFunc is the handler signature for AfterFileWrite.
type FileWriteFunc func(FileWriteEvent) FileWriteVerdict

// AfterFileWrite registers a unified handler for post-file-edit events on all platforms:
//   - Claude Code   → "claude-after-file-write"  (PostToolUse for Write/Edit tools)
//   - Cursor        → "cursor-after-file-edit"   (fire-and-forget)
//   - Windsurf      → "windsurf-post-write-code" (fire-and-forget)
//   - Factory Droid → "droid-after-file-write"   (PostToolUse for Write/Edit tools)
//   - Gemini CLI    → "gemini-after-file-tool"   (AfterTool for Write/Edit tools)
func AfterFileWrite(fn FileWriteFunc) {
	// Claude Code — PostToolUse filtered to Write and Edit tools
	dispatch.AddRoute("claude-after-file-write", func() {
		dispatch.Process(func(ev claude.PostToolUseEvent) claude.PostToolUseResult {
			if ev.ToolName != "Write" && ev.ToolName != "Edit" && ev.ToolName != "MultiEdit" {
				return claude.AcknowledgeToolUse()
			}
			changes := claudeWriteChanges(ev.ToolName, ev.ToolInput)
			verdict := fn(FileWriteEvent{
				Agent: AgentClaude, SessionID: ev.SessionID,
				FilePath: claudeFilePath(ev.ToolInput), Changes: changes, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if verdict.Reject {
				return claude.RejectToolResult(verdict.Feedback)
			}
			if verdict.Footnote != "" {
				return claude.AddToolContext(verdict.Footnote)
			}
			return claude.AcknowledgeToolUse()
		})
	})

	// Cursor — afterFileEdit (fire-and-forget)
	dispatch.AddRoute("cursor-after-file-edit", func() {
		dispatch.Process(func(ev cursor.FileEditEvent) cursor.FileEditResult {
			changes := make([]FileDiff, len(ev.Edits))
			for i, e := range ev.Edits {
				changes[i] = FileDiff{Before: e.OldText, After: e.NewText}
			}
			fn(FileWriteEvent{
				Agent: AgentCursor, SessionID: ev.ConversationID,
				FilePath: ev.FilePath, Changes: changes, Raw: &ev,
			})
			return cursor.FileEditResult{}
		})
	})

	// Windsurf — post_write_code (fire-and-forget)
	dispatch.AddRoute("windsurf-post-write-code", func() {
		dispatch.Process(func(ev windsurf.PostWriteCodeEvent) windsurf.PostWriteCodeResult {
			changes := make([]FileDiff, len(ev.ToolInfo.Edits))
			for i, e := range ev.ToolInfo.Edits {
				changes[i] = FileDiff{Before: e.OldText, After: e.NewText}
			}
			fn(FileWriteEvent{
				Agent: AgentWindsurf, SessionID: ev.TrajectoryID,
				FilePath: ev.ToolInfo.FilePath, Changes: changes, Raw: &ev,
			})
			return windsurf.PostWriteCodeResult{}
		})
	})

	// Factory Droid — PostToolUse filtered to Write and Edit tools
	dispatch.AddRoute("droid-after-file-write", func() {
		dispatch.Process(func(ev droid.PostToolUseEvent) droid.PostToolUseResult {
			if ev.ToolName != "Write" && ev.ToolName != "Edit" && ev.ToolName != "MultiEdit" {
				return droid.AcknowledgeToolUse()
			}
			changes := claudeWriteChanges(ev.ToolName, ev.ToolInput) // same schema as Claude
			verdict := fn(FileWriteEvent{
				Agent: AgentDroid, SessionID: ev.SessionID,
				FilePath: claudeFilePath(ev.ToolInput), Changes: changes, WorkDir: ev.WorkDir,
				Raw: &ev,
			})
			if verdict.Reject {
				return droid.RejectToolResult(verdict.Feedback)
			}
			if verdict.Footnote != "" {
				return droid.AddToolContext(verdict.Footnote)
			}
			return droid.AcknowledgeToolUse()
		})
	})

	// Gemini CLI — AfterTool filtered to Write/Edit tools
	dispatch.AddRoute("gemini-after-file-tool", func() {
		dispatch.Process(func(ev gemini.AfterToolEvent) gemini.AfterToolResult {
			if ev.ToolName != "write_file" && ev.ToolName != "replace_in_file" &&
				ev.ToolName != "Write" && ev.ToolName != "Edit" {
				return gemini.AcknowledgeToolCall()
			}
			var toolInputFields struct {
				FilePath string `json:"file_path"`
				Path     string `json:"path"`
			}
			json.Unmarshal(ev.ToolInput, &toolInputFields) //nolint:errcheck
			filePath := toolInputFields.FilePath
			if filePath == "" {
				filePath = toolInputFields.Path
			}
			verdict := fn(FileWriteEvent{
				Agent: AgentGemini, SessionID: ev.SessionID,
				FilePath: filePath, WorkDir: ev.WorkDir, Raw: &ev,
			})
			if verdict.Footnote != "" {
				return gemini.AddToolAnnotation(verdict.Footnote)
			}
			return gemini.AcknowledgeToolCall()
		})
	})
}

// claudeFilePath extracts the file_path field from Claude/Droid tool input JSON.
func claudeFilePath(input json.RawMessage) string {
	var v struct {
		FilePath string `json:"file_path"`
	}
	json.Unmarshal(input, &v) //nolint:errcheck
	return v.FilePath
}

// claudeWriteChanges builds FileDiff entries from Claude Code / Factory Droid tool input JSON.
// Both agents use the same tool input schema.
func claudeWriteChanges(toolName string, input json.RawMessage) []FileDiff {
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

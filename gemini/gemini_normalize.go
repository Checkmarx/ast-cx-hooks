package gemini

import (
	"encoding/json"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// FileChanges extracts proposed before/after content from a Gemini BeforeTool
// write_file/replace payload. Gemini has no native diff channel; rebuilding
// Changes lets ASCA guardrails scan proposed content before the write lands.
func FileChanges(toolName string, input json.RawMessage) []hookcore.FileDiff {
	var v struct {
		Content   string `json:"content"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	if json.Unmarshal(input, &v) != nil {
		return nil
	}
	if toolName == "replace" && (v.OldString != "" || v.NewString != "") {
		return []hookcore.FileDiff{{Before: v.OldString, After: v.NewString}}
	}
	if v.Content != "" {
		return []hookcore.FileDiff{{Before: "", After: v.Content}}
	}
	return nil
}

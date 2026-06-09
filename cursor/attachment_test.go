package cursor_test

import (
	"encoding/json"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
)

// TestAttachmentFilePathSnakeCase guards the casing fix: Cursor sends attachment
// paths as snake_case file_path, not camelCase filePath.
func TestAttachmentFilePathSnakeCase(t *testing.T) {
	raw := `{"prompt":"hi","attachments":[{"type":"file","file_path":"/repo/secret.env"}]}`
	var ev cursor.PromptPreEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(ev.Attachments) != 1 {
		t.Fatalf("attachments: %+v", ev.Attachments)
	}
	if ev.Attachments[0].FilePath != "/repo/secret.env" {
		t.Fatalf("FilePath=%q, want /repo/secret.env (snake_case file_path)", ev.Attachments[0].FilePath)
	}
}

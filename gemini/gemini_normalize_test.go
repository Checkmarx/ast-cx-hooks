package gemini_test

import (
	"encoding/json"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks/gemini"
)

func TestFileChanges_WriteFile(t *testing.T) {
	ch := gemini.FileChanges("write_file", json.RawMessage(`{"file_path":"src/Demo.java","content":"class Demo {}"}`))
	if len(ch) != 1 || ch[0].Before != "" || ch[0].After != "class Demo {}" {
		t.Fatalf("unexpected Changes: %+v", ch)
	}
}

func TestFileChanges_Replace(t *testing.T) {
	ch := gemini.FileChanges("replace", json.RawMessage(`{"file_path":"app.py","old_string":"x=1","new_string":"x=2"}`))
	if len(ch) != 1 || ch[0].Before != "x=1" || ch[0].After != "x=2" {
		t.Fatalf("unexpected Changes: %+v", ch)
	}
}

func TestFileChanges_InvalidInput(t *testing.T) {
	if ch := gemini.FileChanges("write_file", json.RawMessage(`not-json`)); ch != nil {
		t.Fatalf("expected nil Changes for invalid input, got %+v", ch)
	}
}

func TestFileChanges_EmptyContent(t *testing.T) {
	if ch := gemini.FileChanges("write_file", json.RawMessage(`{"file_path":"x.txt"}`)); ch != nil {
		t.Fatalf("expected nil Changes for empty content, got %+v", ch)
	}
}

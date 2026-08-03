package cursor

import (
	"encoding/json"
	"testing"
)

func TestMillisecondsUnmarshalInteger(t *testing.T) {
	var m Milliseconds
	if err := json.Unmarshal([]byte(`42`), &m); err != nil {
		t.Fatalf("unmarshal int: %v", err)
	}
	if m != 42 {
		t.Fatalf("got %d, want 42", m)
	}
}

func TestMillisecondsUnmarshalFloat(t *testing.T) {
	var m Milliseconds
	if err := json.Unmarshal([]byte(`2850.456`), &m); err != nil {
		t.Fatalf("unmarshal float: %v", err)
	}
	if m != 2850 {
		t.Fatalf("got %d, want 2850 (rounded)", m)
	}
}

func TestUnmarshalToolPostEventFloatDuration(t *testing.T) {
	payload := `{
		"hook_event_name": "postToolUse",
		"tool_name": "Write",
		"tool_input": {"file_path": "a.txt", "content": "x"},
		"tool_output": "{}",
		"duration": 2850.456
	}`
	var e ToolPostEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Duration != 2850 {
		t.Fatalf("Duration = %d, want 2850", e.Duration)
	}
}

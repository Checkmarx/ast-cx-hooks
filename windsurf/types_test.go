package windsurf_test

import (
	"encoding/json"
	"testing"

	"github.com/Checkmarx/ast-cx-hooks/windsurf"
)

// TestEventBaseModelName verifies the model_name common field parses on every payload.
func TestEventBaseModelName(t *testing.T) {
	raw := `{
		"agent_action_name":"pre_run_command",
		"trajectory_id":"traj-1",
		"execution_id":"exec-1",
		"timestamp":"2026-06-01T00:00:00Z",
		"model_name":"Claude Sonnet 4",
		"tool_info":{"command_line":"ls","cwd":"/repo"}
	}`
	var ev windsurf.PreRunCommandEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ModelName != "Claude Sonnet 4" {
		t.Fatalf("ModelName=%q, want Claude Sonnet 4", ev.ModelName)
	}
	if ev.ToolInfo.CommandLine != "ls" {
		t.Fatalf("CommandLine=%q", ev.ToolInfo.CommandLine)
	}
}

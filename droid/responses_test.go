package droid_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Checkmarx/ast-cx-hooks/droid"
)

func TestSubagentStopResponses(t *testing.T) {
	if r := droid.LetSubagentStop(); r.Proceed == nil || !*r.Proceed {
		t.Fatal("LetSubagentStop should set continue=true")
	}
	b := droid.HaltSubagent("keep going")
	if b.Decision != "block" || b.Reason != "keep going" {
		t.Fatalf("HaltSubagent: %+v", b)
	}
}

func TestNotificationResponses(t *testing.T) {
	if r := droid.AcknowledgeNotification(); r.SystemNote != "" {
		t.Fatal("AcknowledgeNotification should be empty")
	}
	n := droid.NotifyWithSystemMessage("heads up")
	if n.SystemNote != "heads up" {
		t.Fatalf("NotifyWithSystemMessage: SystemNote=%q", n.SystemNote)
	}
}

func TestApproveToolUseWithInput(t *testing.T) {
	r := droid.ApproveToolUseWithInput(json.RawMessage(`{"command":"ls -la"}`))
	if r.Details == nil || r.Details.Decision != "allow" {
		t.Fatalf("ApproveToolUseWithInput should allow: %+v", r.Details)
	}
	b, _ := json.Marshal(r)
	if out := string(b); !strings.Contains(out, `"updatedInput"`) || !strings.Contains(out, `"ls -la"`) {
		t.Fatalf("updatedInput not emitted: %s", out)
	}
}

func TestRejectToolResultWithContext(t *testing.T) {
	r := droid.RejectToolResultWithContext("blocked: secret detected", "run the cx-secret-remediation skill")
	if r.Decision != "block" || r.Reason != "blocked: secret detected" {
		t.Fatalf("RejectToolResultWithContext should block with reason: decision=%q reason=%q", r.Decision, r.Reason)
	}
	if r.Details == nil || r.Details.ExtraContext != "run the cx-secret-remediation skill" {
		t.Fatalf("RejectToolResultWithContext should carry additionalContext: %+v", r.Details)
	}
	b, _ := json.Marshal(r)
	out := string(b)
	if !strings.Contains(out, `"decision":"block"`) {
		t.Fatalf("decision not emitted: %s", out)
	}
	if !strings.Contains(out, `"reason":"blocked: secret detected"`) {
		t.Fatalf("reason not emitted: %s", out)
	}
	if !strings.Contains(out, `"additionalContext":"run the cx-secret-remediation skill"`) {
		t.Fatalf("additionalContext not emitted: %s", out)
	}
}

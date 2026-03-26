package cursor_test

import (
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
)

func TestPermissionHelpers(t *testing.T) {
	p := cursor.Permit()
	if p.Permission != "allow" {
		t.Fatalf("Permit: got %q", p.Permission)
	}

	f := cursor.Forbid("not ok", "denied")
	if f.Permission != "deny" {
		t.Fatalf("Forbid: got %q", f.Permission)
	}
	if f.UserNote != "not ok" {
		t.Fatalf("Forbid: UserNote=%q", f.UserNote)
	}

	ask := cursor.RequestConfirmation("please confirm", "agent confirm")
	if ask.Permission != "ask" {
		t.Fatalf("RequestConfirmation: got %q", ask.Permission)
	}
}

func TestStopHelpers(t *testing.T) {
	s := cursor.LetStop()
	if s.FollowupText != "" {
		t.Fatal("LetStop should have no followup")
	}

	f := cursor.SendFollowup("keep going")
	if f.FollowupText != "keep going" {
		t.Fatalf("SendFollowup: got %q", f.FollowupText)
	}
}

func TestPromptHelpers(t *testing.T) {
	a := cursor.AcceptPrompt()
	if !a.Continue {
		t.Fatal("AcceptPrompt should set Continue=true")
	}

	b := cursor.BlockPrompt("nope")
	if b.Continue {
		t.Fatal("BlockPrompt should set Continue=false")
	}
	if b.UserNote != "nope" {
		t.Fatalf("BlockPrompt: UserNote=%q", b.UserNote)
	}
}

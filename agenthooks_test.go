package agenthooks_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks"
)

func TestAddRouteAndDispatch(t *testing.T) {
	agenthooks.ClearRoutes()
	called := false
	agenthooks.AddRoute("test-cmd", func() { called = true })

	os.Args = []string{"myhook", "test-cmd"}
	agenthooks.Dispatch()

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestProcessReadsStdinAndWritesStdout(t *testing.T) {
	type Input struct {
		Value string `json:"value"`
	}
	type Output struct {
		Result string `json:"result"`
	}

	// Write JSON to a temp file and point stdin to it.
	f, err := os.CreateTemp("", "agenthooks-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	json.NewEncoder(f).Encode(Input{Value: "hello"})
	f.Seek(0, 0)

	origStdin := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = origStdin }()

	// Capture stdout.
	outFile, err := os.CreateTemp("", "agenthooks-out-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outFile.Name())
	origStdout := os.Stdout
	os.Stdout = outFile
	defer func() { os.Stdout = origStdout }()

	agenthooks.Process(func(in Input) Output {
		return Output{Result: "got:" + in.Value}
	})

	outFile.Seek(0, 0)
	var out Output
	if err := json.NewDecoder(outFile).Decode(&out); err != nil {
		t.Fatalf("decoding output: %v", err)
	}
	if out.Result != "got:hello" {
		t.Fatalf("unexpected result: %q", out.Result)
	}
}

func TestClearRoutes(t *testing.T) {
	agenthooks.ClearRoutes()
	agenthooks.AddRoute("x", func() {})
	agenthooks.ClearRoutes()

	// After clearing, Dispatch should exit 1. We can't test os.Exit directly,
	// so just verify AddRoute after ClearRoutes works without panic.
	agenthooks.AddRoute("y", func() {})
}

func TestUnifiedAgentIDConstants(t *testing.T) {
	ids := []agenthooks.AgentID{
		agenthooks.AgentClaude,
		agenthooks.AgentCursor,
		agenthooks.AgentWindsurf,
		agenthooks.AgentDroid,
		agenthooks.AgentGemini,
	}
	seen := map[agenthooks.AgentID]bool{}
	for _, id := range ids {
		if id == "" {
			t.Fatal("AgentID must not be empty")
		}
		if seen[id] {
			t.Fatalf("duplicate AgentID: %q", id)
		}
		seen[id] = true
	}
}

func TestIsLooping(t *testing.T) {
	tests := []struct {
		name   string
		event  agenthooks.AgentIdleEvent
		expect bool
	}{
		{
			name:   "claude: repeat=true means looping",
			event:  agenthooks.AgentIdleEvent{Agent: agenthooks.AgentClaude, IsRepeat: true},
			expect: true,
		},
		{
			name:   "claude: repeat=false means not looping",
			event:  agenthooks.AgentIdleEvent{Agent: agenthooks.AgentClaude, IsRepeat: false},
			expect: false,
		},
		{
			name:   "droid: repeat=true means looping",
			event:  agenthooks.AgentIdleEvent{Agent: agenthooks.AgentDroid, IsRepeat: true},
			expect: true,
		},
		{
			name:   "gemini: repeat=true means looping",
			event:  agenthooks.AgentIdleEvent{Agent: agenthooks.AgentGemini, IsRepeat: true},
			expect: true,
		},
		{
			name:   "cursor: loop count < 3 is not looping",
			event:  agenthooks.AgentIdleEvent{Agent: agenthooks.AgentCursor, AutoRetryCount: 2},
			expect: false,
		},
		{
			name:   "cursor: loop count >= 3 is looping",
			event:  agenthooks.AgentIdleEvent{Agent: agenthooks.AgentCursor, AutoRetryCount: 3},
			expect: true,
		},
		{
			name:   "windsurf: never looping (fire-and-forget)",
			event:  agenthooks.AgentIdleEvent{Agent: agenthooks.AgentWindsurf, IsRepeat: true},
			expect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.event.IsLooping(); got != tc.expect {
				t.Fatalf("IsLooping() = %v, want %v", got, tc.expect)
			}
		})
	}
}

func TestVerdictHelpers(t *testing.T) {
	r := agenthooks.Resume()
	if !r.Proceed {
		t.Fatal("Resume() should have Proceed=true")
	}

	i := agenthooks.Interrupt("do more work")
	if i.Proceed {
		t.Fatal("Interrupt() should have Proceed=false")
	}
	if i.Feedback != "do more work" {
		t.Fatalf("Interrupt() feedback: got %q", i.Feedback)
	}

	a := agenthooks.Allow()
	if !a.Permit {
		t.Fatal("Allow() should have Permit=true")
	}

	d := agenthooks.Deny("blocked")
	if d.Permit {
		t.Fatal("Deny() should have Permit=false")
	}

	ap := agenthooks.AcceptPrompt()
	if !ap.Accept {
		t.Fatal("AcceptPrompt() should have Accept=true")
	}

	rp := agenthooks.RejectPrompt("no")
	if rp.Accept {
		t.Fatal("RejectPrompt() should have Accept=false")
	}
}

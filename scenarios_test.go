package agenthooks_test

import (
	"os"
	"strings"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

// dispatchToolCall runs the claude-pre-tool-use route with an optional scenario
// arg and returns stdout. Routes/scenarios must be registered before calling.
func dispatchToolCall(t *testing.T, scenario string) string {
	t.Helper()
	stdin := `{"session_id":"s","cwd":"/r","tool_name":"Bash","tool_input":{"command":"ls"}}`
	stdoutBuf := pipeStdio(t, stdin)

	origArgs := os.Args
	if scenario == "" {
		os.Args = []string{"hook", "claude-pre-tool-use"}
	} else {
		os.Args = []string{"hook", "claude-pre-tool-use", scenario}
	}
	defer func() { os.Args = origArgs }()

	agenthooks.Dispatch()
	return stdoutBuf()
}

// TestScenarioDispatchByArg verifies that two teams can share one hook, selected
// by the scenario arg the caller passes, with a default fallback.
func TestScenarioDispatchByArg(t *testing.T) {
	agenthooks.ClearRoutes()

	agenthooks.BeforeToolCall(func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("default-policy")
	})
	agenthooks.BeforeToolCallScenario("phoenix", func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("phoenix-policy")
	})
	agenthooks.BeforeToolCallScenario("cypher", func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.AllowWithNote("cypher-ok")
	})

	cases := []struct{ scenario, want string }{
		{"phoenix", "phoenix-policy"}, // matched scenario
		{"cypher", "cypher-ok"},       // a different team's logic
		{"", "default-policy"},        // no key → default
		{"unknown", "default-policy"}, // unregistered key → default
	}
	for _, tc := range cases {
		t.Run("scenario="+tc.scenario, func(t *testing.T) {
			if out := dispatchToolCall(t, tc.scenario); !strings.Contains(out, tc.want) {
				t.Fatalf("scenario %q: stdout %q missing %q", tc.scenario, out, tc.want)
			}
		})
	}
}

// TestScenarioContentSelector verifies a custom selector can pick the scenario
// from the event content (here: the working directory) with no arg passed.
func TestScenarioContentSelector(t *testing.T) {
	agenthooks.ClearRoutes()
	agenthooks.UseScenarioSelector(func(m agenthooks.ScenarioMeta) string {
		if strings.Contains(m.WorkDir, "phoenix") {
			return "phoenix"
		}
		return ""
	})
	agenthooks.BeforeToolCall(func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("default-policy")
	})
	agenthooks.BeforeToolCallScenario("phoenix", func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("phoenix-by-cwd")
	})

	stdin := `{"session_id":"s","cwd":"/repos/phoenix-svc","tool_name":"Bash","tool_input":{"command":"ls"}}`
	stdoutBuf := pipeStdio(t, stdin)
	origArgs := os.Args
	os.Args = []string{"hook", "claude-pre-tool-use"} // no scenario arg; selector uses cwd
	defer func() { os.Args = origArgs }()

	agenthooks.Dispatch()
	if out := stdoutBuf(); !strings.Contains(out, "phoenix-by-cwd") {
		t.Fatalf("content selector should route to phoenix by cwd, got: %s", out)
	}
}

// TestScenarioFromEnv verifies AGENTHOOKS_SCENARIO selects the scenario when no
// arg is passed (the default selector's env fallback).
func TestScenarioFromEnv(t *testing.T) {
	agenthooks.ClearRoutes()
	agenthooks.BeforeToolCall(func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("default-policy")
	})
	agenthooks.BeforeToolCallScenario("phoenix", func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("phoenix-by-env")
	})

	t.Setenv("AGENTHOOKS_SCENARIO", "phoenix")
	if out := dispatchToolCall(t, ""); !strings.Contains(out, "phoenix-by-env") {
		t.Fatalf("env var should select phoenix, got: %s", out)
	}
}

// TestScenarioAcrossAllHooks smoke-tests that every hook category exposes a
// working <Hook>Scenario registration (routes register and dispatch by key).
func TestScenarioAcrossAllHooks(t *testing.T) {
	agenthooks.ClearRoutes()

	agenthooks.WhenAgentIdleScenario("a", func(agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Interrupt("idle-a") })
	agenthooks.WhenSubagentIdleScenario("a", func(agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Interrupt("sub-a") })
	agenthooks.BeforeToolCallScenario("a", func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Deny("tool-a") })
	agenthooks.AfterToolFailureScenario("a", func(agenthooks.ToolFailureEvent) agenthooks.ToolFailureVerdict {
		return agenthooks.AnnotateFailure("fail-a")
	})
	agenthooks.AfterFileWriteScenario("a", func(agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
		return agenthooks.AnnotateWrite("write-a")
	})
	agenthooks.BeforeFileReadScenario("a", func(agenthooks.FileReadEvent) agenthooks.FileReadVerdict { return agenthooks.DenyRead("read-a") })
	agenthooks.BeforePromptScenario("a", func(agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.RejectPrompt("prompt-a") })

	registered := map[string]bool{}
	for _, r := range agenthooks.RouteNames() {
		registered[r] = true
	}
	for _, route := range []string{
		"claude-stop", "claude-subagent-stop", "claude-pre-tool-use",
		"claude-post-tool-use-failure", "claude-after-file-write",
		"cursor-before-read-file", "claude-user-prompt-submit",
	} {
		if !registered[route] {
			t.Errorf("scenario registration did not wire route %q", route)
		}
	}

	// Spot-check one route dispatches to the scenario via the arg.
	cases := []struct{ route, scenario, want string }{
		{"claude-pre-tool-use", "a", "tool-a"},
		{"claude-user-prompt-submit", "a", "prompt-a"},
		{"cursor-before-read-file", "a", "read-a"},
	}
	stdins := map[string]string{
		"claude-pre-tool-use":       `{"session_id":"s","tool_name":"Bash","tool_input":{"command":"ls"}}`,
		"claude-user-prompt-submit": `{"session_id":"s","hook_event_name":"UserPromptSubmit","prompt":"hi"}`,
		"cursor-before-read-file":   `{"conversation_id":"c","file_path":"/r/x"}`,
	}
	for _, tc := range cases {
		t.Run(tc.route, func(t *testing.T) {
			stdoutBuf := pipeStdio(t, stdins[tc.route])
			origArgs := os.Args
			os.Args = []string{"hook", tc.route, tc.scenario}
			defer func() { os.Args = origArgs }()
			agenthooks.Dispatch()
			if out := stdoutBuf(); !strings.Contains(out, tc.want) {
				t.Fatalf("%s scenario %q: stdout %q missing %q", tc.route, tc.scenario, out, tc.want)
			}
		})
	}
}

// TestScenarioFailsClosedWithoutDefault verifies that a gating hook with only
// named scenarios (no default) does NOT silently allow when no key matches — it
// must fail closed (deny).
func TestScenarioFailsClosedWithoutDefault(t *testing.T) {
	agenthooks.ClearRoutes()
	agenthooks.BeforeToolCallScenario("phoenix", func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("phoenix-policy")
	})
	out := dispatchToolCall(t, "") // no scenario key, no default registered
	if !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("tool-call gate must fail closed (deny) when no handler matches, got: %s", out)
	}
	if !strings.Contains(out, "failing closed") {
		t.Fatalf("expected the fail-closed fallback reason, got: %s", out)
	}
}

// TestScenarioArgBeatsEnv verifies the CLI scenario arg takes precedence over
// AGENTHOOKS_SCENARIO.
func TestScenarioArgBeatsEnv(t *testing.T) {
	agenthooks.ClearRoutes()
	agenthooks.BeforeToolCallScenario("phoenix", func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("phoenix-fired")
	})
	agenthooks.BeforeToolCallScenario("cypher", func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("cypher-fired")
	})
	t.Setenv("AGENTHOOKS_SCENARIO", "phoenix")
	out := dispatchToolCall(t, "cypher") // arg cypher must beat env phoenix
	if !strings.Contains(out, "cypher-fired") || strings.Contains(out, "phoenix-fired") {
		t.Fatalf("CLI arg should win over env var, got: %s", out)
	}
}

// TestScenarioCustomSelectorUnknownKey verifies a custom selector returning an
// unregistered key falls back to the default handler.
func TestScenarioCustomSelectorUnknownKey(t *testing.T) {
	agenthooks.ClearRoutes()
	agenthooks.UseScenarioSelector(func(agenthooks.ScenarioMeta) string { return "ghost" })
	agenthooks.BeforeToolCall(func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
		return agenthooks.Deny("default-fired")
	})
	if out := dispatchToolCall(t, ""); !strings.Contains(out, "default-fired") {
		t.Fatalf("unknown selector key should fall back to the default handler, got: %s", out)
	}
}

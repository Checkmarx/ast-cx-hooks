package agenthooks_test

import (
	"testing"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
)

// registerAllUnifiedHandlers wires every unified handler with no-op callbacks so
// the full set of routes is present in the registry.
func registerAllUnifiedHandlers() {
	agenthooks.ClearRoutes()
	agenthooks.WhenAgentIdle(func(agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.BeforeToolCall(func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Allow() })
	agenthooks.AfterFileWrite(func(agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict { return agenthooks.AcceptWrite() })
	agenthooks.BeforePrompt(func(agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.AcceptPrompt() })
	agenthooks.WhenSubagentIdle(func(agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.AfterToolFailure(func(agenthooks.ToolFailureEvent) agenthooks.ToolFailureVerdict { return agenthooks.AcknowledgeFailure() })
	agenthooks.BeforeFileRead(func(agenthooks.FileReadEvent) agenthooks.FileReadVerdict { return agenthooks.AllowRead() })
}

// TestCatalogRoutesAreRegistered enforces the catalog↔registration invariant:
// every route the install command writes must be one a unified handler registers.
// This is the guarantee the route Catalog exists to provide.
func TestCatalogRoutesAreRegistered(t *testing.T) {
	registerAllUnifiedHandlers()

	registered := map[string]bool{}
	for _, r := range agenthooks.RouteNames() {
		registered[r] = true
	}

	seen := map[string]bool{}
	for _, e := range agenthooks.Catalog {
		if seen[e.Route] {
			t.Fatalf("duplicate catalog route %q", e.Route)
		}
		seen[e.Route] = true

		if e.Agent == "" || e.SettingsRel == "" || e.EventKey == "" || e.Style == "" {
			t.Fatalf("catalog entry for %q has empty fields: %+v", e.Route, e)
		}
		if !registered[e.Route] {
			t.Errorf("catalog route %q is not registered by any unified handler", e.Route)
		}
	}
}

package agenthooks_test

import (
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

// registerAllUnifiedHandlers wires every unified handler with no-op callbacks so
// the full set of routes is present in the registry.
func registerAllUnifiedHandlers() {
	agenthooks.ClearRoutes()
	agenthooks.WhenAgentIdle(func(agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.BeforeToolCall(func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Allow() })
	agenthooks.AfterFileWrite(func(agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict { return agenthooks.AcceptWrite() })
	agenthooks.BeforeFileEdit(func(agenthooks.FileEditEvent) agenthooks.FileEditVerdict { return agenthooks.AcceptEdit() })
	agenthooks.BeforePrompt(func(agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.AcceptPrompt() })
	agenthooks.WhenSubagentIdle(func(agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.AfterToolFailure(func(agenthooks.ToolFailureEvent) agenthooks.ToolFailureVerdict {
		return agenthooks.AcknowledgeFailure()
	})
	agenthooks.BeforeFileRead(func(agenthooks.FileReadEvent) agenthooks.FileReadVerdict { return agenthooks.AllowRead() })
}

// Design note (route dual-truth). A route name is declared in two places: the
// Catalog (which carries install-only metadata — settings path, event key, style)
// and the platform adapter (which returns it for dispatch). Unifying them into one
// source was considered and rejected: the 7 dispatch registries are typed
// scenarioReg[E,V] with distinct event/verdict generics, so a single data table
// driving both dispatch and install would force any-erasure of those generics —
// trading compile-time type safety for one list. The two invariant tests below
// (forward + reverse) bridge the dual-truth cheaply instead: a route can never be
// installed-but-unregistered, nor registered-but-not-installed, without CI failing.

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

// intentionallyUninstalledRoutes are registered routes that the Catalog deliberately
// omits because the platform's hooks are not configured via a home-directory settings
// file. VS Code Copilot hooks are project-scoped (.github/hooks), so install skips them.
var intentionallyUninstalledRoutes = map[string]bool{
	"copilot-stop":               true,
	"copilot-subagent-stop":      true,
	"copilot-pre-tool-use":       true,
	"copilot-pre-file-write":     true,
	"copilot-after-file-write":   true,
	"copilot-user-prompt-submit": true,
}

// TestRegisteredRoutesAreCataloged enforces the REVERSE direction: every registered
// route must either be in the Catalog (so install writes it) or be an explicitly
// allow-listed intentional omission. This fails CI if a new adapter route is added
// without a catalog entry (which would silently never be installed).
func TestRegisteredRoutesAreCataloged(t *testing.T) {
	registerAllUnifiedHandlers()

	inCatalog := map[string]bool{}
	for _, e := range agenthooks.Catalog {
		inCatalog[e.Route] = true
	}

	for _, route := range agenthooks.RouteNames() {
		if !inCatalog[route] && !intentionallyUninstalledRoutes[route] {
			t.Errorf("registered route %q is neither in the Catalog nor allow-listed as intentionally uninstalled", route)
		}
	}
}

// Package agenthooks provides a framework for building hooks for AI coding agents.
//
// It supports Claude Code, Cursor, Windsurf Cascade, Factory Droid, Gemini CLI, and
// VS Code Copilot (Preview) through a single unified API plus platform-specific
// packages for advanced use. (Copilot hooks are project-scoped, so `agenthooks
// install` configures the other five; see the README for Copilot setup.)
//
// Quick start with unified handlers (one handler works across all agents):
//
//	package main
//
//	import "github.com/CheckmarxDev/ast-cx-hooks"
//
//	func main() {
//	    agenthooks.WhenAgentIdle(func(e agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
//	        if e.IsLooping() {
//	            return agenthooks.Resume()
//	        }
//	        return agenthooks.Interrupt("Please review changes before finishing.")
//	    })
//	    agenthooks.Dispatch()
//	}
//
// Platform-specific handlers for advanced use:
//
//	agenthooks.AddRoute("claude-pre-tool-use", func() {
//	    agenthooks.Process(func(e claude.PreToolUseEvent) claude.PreToolUseResult {
//	        return claude.ApproveToolUse()
//	    })
//	})
package agenthooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// RouteFunc is the type for handlers registered via AddRoute.
type RouteFunc func()

var routes = map[string]RouteFunc{}

// AddRoute registers fn under the given command name.
// When Dispatch is called and os.Args[1] matches name, fn is invoked.
func AddRoute(name string, fn RouteFunc) {
	routes[name] = fn
}

// Dispatch selects and runs the handler whose name matches os.Args[1].
// If os.Args[1] is absent, the binary name is used instead.
// Exits with code 1 if no matching handler is found.
func Dispatch() {
	name := resolveRouteName()
	if fn, ok := routes[name]; ok {
		fn()
		return
	}
	fmt.Fprintf(os.Stderr, "agenthooks: no handler registered for %q\n", name)
	fmt.Fprintln(os.Stderr, "available routes:")
	for k := range routes {
		fmt.Fprintf(os.Stderr, "  %s\n", k)
	}
	os.Exit(1)
}

// Process reads JSON from stdin, passes it to handler, and writes the result to stdout.
// Any stdin parse error causes a graceful exit (code 0) so a bad payload never blocks an agent.
// The implementation lives in internal/hookcore so platform adapters share it.
func Process[I any, O any](handler func(I) O) { hookcore.Run(handler) }

// ProcessE is like Process but allows the handler to signal a blocking error.
// When handler returns a non-nil error, agenthooks writes the message to stderr
// and exits with code 2, which causes supporting agents to surface the message
// and block the pending action.
func ProcessE[I any, O any](handler func(I) (O, error)) { hookcore.RunE(handler) }

// ClearRoutes removes all registered handlers and scenario state. Intended for use in tests.
func ClearRoutes() {
	routes = map[string]RouteFunc{}
	resetScenarios()
}

// RouteNames returns the names of all currently registered routes, unsorted.
// Intended for inspection and tests (e.g. verifying the route Catalog stays in
// sync with what the unified handlers register).
func RouteNames() []string {
	names := make([]string, 0, len(routes))
	for name := range routes {
		names = append(names, name)
	}
	return names
}

func resolveRouteName() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return filepath.Base(os.Args[0])
}

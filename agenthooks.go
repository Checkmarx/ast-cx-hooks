// Package agenthooks provides a framework for building hooks for AI coding agents.
//
// It supports Claude Code, Cursor, Windsurf Cascade, Factory Droid, and Gemini CLI
// through a single unified API plus platform-specific packages for advanced use.
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

	"github.com/CheckmarxDev/ast-cx-hooks/internal/codec"
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
func Process[I any, O any](handler func(I) O) {
	var in I
	if err := codec.DecodeStdin(&in); err != nil {
		os.Exit(0)
	}
	out := handler(in)
	if err := codec.EncodeStdout(out); err != nil {
		os.Exit(0)
	}
}

// ProcessE is like Process but allows the handler to signal a blocking error.
// When handler returns a non-nil error, agenthooks writes the message to stderr
// and exits with code 2, which causes supporting agents to surface the message
// and block the pending action.
func ProcessE[I any, O any](handler func(I) (O, error)) {
	var in I
	if err := codec.DecodeStdin(&in); err != nil {
		os.Exit(0)
	}
	out, err := handler(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	if err := codec.EncodeStdout(out); err != nil {
		os.Exit(0)
	}
}

// ClearRoutes removes all registered handlers. Intended for use in tests.
func ClearRoutes() {
	routes = map[string]RouteFunc{}
}

func resolveRouteName() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return filepath.Base(os.Args[0])
}

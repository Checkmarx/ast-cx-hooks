// Package windsurf provides types and response helpers for Windsurf Cascade hooks.
//
// Windsurf hooks are configured in hooks.json files at system, user, or workspace
// scope. Pre-hooks can block actions by exiting with code 2; post-hooks are
// informational only. All hooks receive a JSON payload on stdin via a nested
// tool_info object.
//
// See https://docs.windsurf.com/windsurf/cascade/hooks for the official reference.
package windsurf

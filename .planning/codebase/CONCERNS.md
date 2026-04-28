# Codebase Concerns

**Analysis Date:** 2026-04-28

## Error Handling & Robustness

**Silent Failures in Process Functions:**
- Issue: `Process()` and `ProcessE()` silently exit with code 0 on stdin/stdout errors without logging
- Files: `agenthooks.go` (lines 71-72, 94-95)
- Impact: Hooks failing to read input or write output will silently disappear, making debugging extremely difficult. Agents may proceed when hooks intended to block.
- Fix approach: Add stderr logging before exiting (e.g., `fmt.Fprintf(os.Stderr, "codec error: %v\n", err)`) to aid diagnostics without changing exit codes.

**Unvalidated JSON Unmarshaling:**
- Issue: Multiple `json.Unmarshal()` calls are error-suppressed with `//nolint:errcheck`, ignoring parsing failures
- Files: 
  - `unified.go` (lines 480, 614, 654, 668)
  - `cmd/agenthooks/main.go` (line 165)
- Impact: Malformed or missing JSON fields will be silently ignored, leaving zero values. Tool command extraction, file path extraction, and arguments may be empty when expected.
- Fix approach: Check errors from json.Unmarshal and fall back gracefully or log warnings. Populate default values when unmarshaling fails.

**Example fragility** in `unified.go:480`:
```go
json.Unmarshal(ev.ToolInput, &toolInputFields) //nolint:errcheck
filePath := toolInputFields.FilePath
if filePath == "" {
  filePath = toolInputFields.Path
}
```
If ToolInput is invalid JSON or missing, `filePath` will be empty and the hook will process an empty file path silently.

## Security Considerations

**No Input Validation on Tool Arguments:**
- Risk: `ToolArgs` (json.RawMessage) is passed to handlers without validation
- Files: `unified.go` (ToolCallEvent struct, line 193)
- Current mitigation: Handlers are expected to validate in their own logic
- Recommendations: 
  - Document that RawMessage may contain untrusted data
  - Consider providing helpers for safe JSON extraction with defaults
  - Add warnings to godoc about injection risks (e.g., in BeforeToolCall handlers)

**Command String Not Sanitized:**
- Risk: Shell command strings are passed directly from agent input to handlers without any escaping or validation
- Files: `unified.go` (ToolCallEvent.Command field, line 182)
- Current mitigation: Handlers must validate before executing
- Recommendations: Add documentation that Command is untrusted input; provide example validation patterns in docs

**No Rate Limiting or DoS Protection:**
- Risk: Hooks can be invoked rapidly in loops with no backpressure. A malicious agent or user could spam hooks.
- Files: `agenthooks.go` (Dispatch, Process functions)
- Impact: Resource exhaustion possible if hooks perform blocking I/O or external API calls
- Recommendations: Consider adding timeout wrappers or documenting performance expectations

**Global Routes Map Not Thread-Safe:**
- Risk: `routes` map in `agenthooks.go` (line 42) is accessed without locks
- Files: `agenthooks.go` (AddRoute, Dispatch functions)
- Impact: If hooks are registered/cleared concurrently with Dispatch, race conditions are possible
- Recommendations: Use sync.RWMutex to protect routes map if concurrent access is possible (currently not apparent from usage)

## Test Coverage Gaps

**Missing Tests for Platform-Specific Implementations:**
- What's not tested: Droid, Gemini, and Windsurf response builders and type conversions
- Files: `droid/responses.go`, `droid/responses_test.go` (missing tests), `gemini/responses.go` (no test file), `windsurf/responses.go` (no test file)
- Risk: Platform-specific JSON response formatting bugs won't be caught until runtime
- Priority: High — These are integration points with external agents

**No Integration Tests for Unified Handlers:**
- What's not tested: The actual routing and event transformation logic in `unified.go` 
- Files: `unified.go` (677 lines, ~0% test coverage)
- Risk: Mapping between unified events and platform-specific events (WhenAgentIdle, BeforeToolCall, etc.) is untested. A breaking change to one platform's schema won't be caught.
- Priority: High

**Tool Kind Detection Not Fully Covered:**
- What's not tested: `claudeToolKind()`, `droidToolKind()`, `geminiToolKind()` functions with edge cases
- Files: `unified.go` (lines 609-635)
- Risk: Tools with prefixes like `mcp__` or names like `execute_bash` may be misclassified
- Priority: Medium

**Cursor-Specific Looping Logic Not Tested:**
- What's not tested: The `AutoRetryCount >= 3` threshold in IsLooping for Cursor
- Files: `unified.go` (line 67, tested in agenthooks_test.go but only at boundaries)
- Risk: Off-by-one errors or logic inversions could miss loop detection
- Priority: Medium

## Fragile Areas

**Claude vs. Droid Code Duplication:**
- Files: `unified.go` (lines 609-625 duplicated logic, lines 650-677 duplicated)
- Why fragile: `claudeToolKind()` and `droidToolKind()` are identical, as are `claudeFilePath()` and `droidFilePath()`. Changes to one won't propagate to the other.
- Safe modification: Extract shared logic into a single function (e.g., `standardToolKind()`) and have both call it
- Test coverage: Basic unit tests exist but no regression tests ensuring they stay synchronized

**JSON Field Name Inconsistencies:**
- Why fragile: Different agents use different field names for the same concept (e.g., `cwd` vs. `WorkDir`, `conversation_id` vs `session_id`)
- Files: Throughout platform-specific type definitions (`claude/types.go`, `cursor/types.go`, `droid/types.go`, `gemini/types.go`)
- Safe modification: Document all field mappings in a single reference table. Consider code generation or explicit mapping functions to reduce manual errors.

**BOM Stripping Only in DecodeStdin:**
- Why fragile: UTF-8 BOM stripping is only implemented for Cursor in `codec.go` (lines 20-21)
- Files: `internal/codec/codec.go` (line 20)
- Risk: If other agents introduce BOM in the future, DecodeStdin will fail silently
- Safe modification: Log a warning when BOM is stripped, or add a parameter to control behavior

**Fire-and-Forget Hook Asymmetry:**
- Why fragile: Windsurf `post-cascade-response` is fire-and-forget but logs a warning when Interrupt is used (unified.go:130), while Cursor `stop` actually supports followup
- Files: `unified.go` (lines 123-134, lines 110-121)
- Risk: Developers may write handlers expecting all platforms to support Interrupt equally
- Safe modification: Add a note in the AgentIdleEvent godoc or provide `event.CanInterrupt()` helper to check platform capabilities

## Performance Bottlenecks

**JSON Marshaling/Unmarshaling in Hot Path:**
- Problem: Every hook invocation involves JSON encode/decode via json.Encoder/Decoder
- Files: `internal/codec/codec.go`, called from every Process/ProcessE invocation
- Cause: Standard json library is not optimized for high-throughput scenarios
- Impact: Likely negligible for typical hook usage (one event per agent action), but could accumulate in batch scenarios
- Improvement path: If profiling shows bottleneck, consider caching encoders or switching to faster libraries (e.g., `encoding/json` with code generation, or third-party libraries like `easyjson`)

**No Connection Pooling or Caching:**
- Problem: Each hook invocation is a fresh process with no state carried between calls
- Files: Affects all platform integrations
- Impact: If hooks need to call external services (security policies, audit logs), they'll incur startup cost per invocation
- Improvement path: Document this limitation. Consider providing a library-level caching layer for common operations.

## Scaling Limits

**Single Binary Per Hook Type:**
- Current capacity: One hook binary handles all routes for a single agent
- Scaling issue: A single slow or buggy route will block all other routes on that agent
- Limit: If a handler for `claude-after-file-write` runs slowly, `claude-user-prompt-submit` and `claude-pre-tool-use` will also slow down (they all run in the same process sequentially)
- Scaling path: Support multiple hook binaries per agent (different hook names can invoke different binaries), though this requires additional agent configuration

**No Concurrency in Hook Handlers:**
- Limit: Each hook runs synchronously; agents wait for response
- Impact: If a handler performs blocking I/O (API calls, database queries), the agent is blocked
- Scaling path: Implement timeout wrappers (e.g., context.WithTimeout) or document guidelines for handler latency budgets

## Dependencies at Risk

**Go 1.21 EOL Approaching:**
- Risk: `go.mod` specifies Go 1.21, which reaches EOL in Dec 2024
- Files: `go.mod` (line 3)
- Impact: Security patches will no longer be released for Go 1.21
- Migration plan: Upgrade to Go 1.22 or later (current stable is 1.23+)

**No External Dependencies (stdlib only):**
- Positive: Framework has zero runtime dependencies, reducing supply-chain risk
- Risk: Reimplementing common patterns (e.g., JSON handling, path manipulation) increases maintenance burden
- Consideration: Keep dependencies minimal; add them only when complexity cost exceeds maintenance cost

## Missing Critical Features

**No Hook Chaining or Composition:**
- Problem: Handlers are independent; can't compose multiple policies easily
- Blocks: Complex policies (e.g., "allow if tool is read-only AND not sensitive") require nesting if/else in the handler
- Workaround: Handlers can call utility functions; no framework support for middleware/pipeline

**No Hook Signature Validation:**
- Problem: Platform schemas may change, but handlers have no way to validate they're receiving expected fields
- Blocks: If an agent adds a new field and the handler doesn't use it, there's no way to know the handler is outdated
- Consideration: Add optional schema validation or version negotiation in hook payloads

**No Audit/Logging Framework:**
- Problem: Handlers are responsible for all logging; no standard audit trail
- Blocks: Organizations can't easily enforce compliance logging (e.g., "log all tool denials")
- Consideration: Provide an optional audit sink interface that handlers can use

**Limited Backwards Compatibility Strategy:**
- Problem: No versioning scheme for hook payloads or response formats
- Blocks: If a platform changes its schema, existing hook binaries may break silently
- Consideration: Add an explicit version field to all payloads and document breaking change policy

## Technical Debt

**Hardcoded Agent Platform Constants:**
- Issue: AgentID values (`"claude"`, `"cursor"`, etc.) are hardcoded strings throughout
- Files: `unified.go` (multiple switch statements using string matching)
- Impact: No compile-time checking; typos introduce silent bugs
- Fix approach: Consider using enums or a type-safe registry instead of string-based routing

**Platform-Specific Type Duplication:**
- Issue: Each platform has its own type definitions (StopEvent, FileEditEvent, etc.) with minimal code sharing
- Files: `claude/types.go`, `cursor/types.go`, `droid/types.go`, `gemini/types.go`, `windsurf/types.go`
- Impact: ~1000 LOC of repetitive struct definitions, difficult to maintain consistency
- Fix approach: Consider code generation or a template-based approach to define platform types

**Nolint Suppressions Without Rationale:**
- Issue: Multiple `//nolint:errcheck` directives without comments explaining why errors are ignored
- Files: `unified.go` (lines 480, 614, 654, 668), `cmd/agenthooks/main.go` (line 165)
- Impact: Future maintainers won't understand intentionality; may accidentally "fix" by checking errors
- Fix approach: Add explanatory comments (e.g., `// Tool arguments may be missing; OK to proceed with zero values`)

**No Structured Logging:**
- Issue: Error messages use simple `fmt.Fprintf(os.Stderr, ...)` 
- Files: Throughout, especially `agenthooks.go`, `cmd/agenthooks/main.go`
- Impact: Difficult to parse logs programmatically; no correlation IDs or structured context
- Fix approach: Consider adding optional structured logging via interfaces (slog in Go 1.21+)

## Known Limitations

**Windows Path Handling:**
- Observed: Codec uses `filepath.FromSlash()` in `internal/scaffold/scaffold.go` (line 26) but DecodeStdin may receive paths with forward slashes
- Files: `internal/scaffold/scaffold.go`, `internal/codec/codec.go`
- Impact: Cross-platform consistency for file paths in events; agents may provide forward slashes even on Windows
- Workaround: Handlers should normalize paths with `filepath.Clean()` or use `filepath.ToSlash()` for consistency

**No Support for Hook Disabling or Feature Flags:**
- Limitation: All registered routes must have handlers; can't conditionally disable a hook
- Files: `agenthooks.go` (Dispatch function)
- Impact: To disable a hook, either remove the registration or have the handler return a no-op verdict
- Workaround: Handlers can check environment variables or external config files to decide behavior

---

*Concerns audit: 2026-04-28*

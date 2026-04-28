# Testing Patterns

**Analysis Date:** 2026-04-28

## Test Framework

**Runner:**
- Go's standard library `testing` package
- Native Go testing (no external framework like testify, ginkgo)

**Run Commands:**
```bash
go test ./...              # Run all tests
go test -v ./...           # Verbose mode
go test -run TestName      # Run specific test
go test -count=1 ./...     # Disable test caching
```

**Assertion Library:**
- Standard library only: `if got != want { t.Fatal(...) }`
- Manual assertions without helper library
- No assertions on error messages; focus on error presence/absence

## Test File Organization

**Location:** Co-located with implementation
- Test files in same directory as implementation: `*_test.go`
- Examples:
  - `agenthooks.go` → `agenthooks_test.go` (root package)
  - `claude/responses.go` → `claude/responses_test.go`
  - `cursor/responses.go` → `cursor/responses_test.go`
  - `internal/codec/codec.go` → `internal/codec/codec_test.go`

**Naming:**
- Test package uses `_test` suffix: `package agenthooks_test`, `package claude_test`
- This tests the public API (what external callers see)
- Functions: `func TestFeature(t *testing.T)`

**Current test files (4 total):**
1. `agenthooks_test.go` - Core routing and unified handler logic
2. `claude/responses_test.go` - Claude response helpers
3. `cursor/responses_test.go` - Cursor response helpers
4. `internal/codec/codec_test.go` - JSON encoding/decoding

## Test Structure

**Typical test pattern:**
```go
func TestFeatureX(t *testing.T) {
    // Arrange: set up test data
    thing := setupThing()
    
    // Act: call function
    result := doSomething(thing)
    
    // Assert: check result
    if result != expected {
        t.Fatalf("got %v, want %v", result, expected)
    }
}
```

**Example from `agenthooks_test.go`:**
```go
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
```

## Subtable Testing Pattern

**Used for parametrized tests:**
```go
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
            name:   "cursor: loop count >= 3 is looping",
            event:  agenthooks.AgentIdleEvent{Agent: agenthooks.AgentCursor, AutoRetryCount: 3},
            expect: true,
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
```

**Key aspects:**
- Anonymous struct slice: `[]struct { name string; event ...; expect ... }`
- Each case has descriptive `name` field for test identification
- Test runs via `t.Run(tc.name, func...)` for isolated subtest reporting
- Assertions use `tc.` prefix for test case data

## Test Fixture Patterns

**Stdin/Stdout mocking:**
Used extensively in `agenthooks_test.go` and `internal/codec/codec_test.go`

**Pattern: Temp file for stdin simulation:**
```go
f, err := os.CreateTemp("", "agenthooks-test-*.json")
if err != nil {
    t.Fatal(err)
}
defer os.Remove(f.Name())

// Write test data to file
json.NewEncoder(f).Encode(Input{Value: "hello"})
f.Seek(0, 0)

// Redirect stdin
origStdin := os.Stdin
os.Stdin = f
defer func() { os.Stdin = origStdin }()

// Now call code that reads from stdin
agenthooks.Process(func(in Input) Output {
    return Output{Result: "got:" + in.Value}
})

// Verify result
outFile.Seek(0, 0)
var out Output
json.NewDecoder(outFile).Decode(&out)
```

**What's being tested:**
- JSON deserialization from stdin
- Function logic
- JSON serialization to stdout
- Proper cleanup and restoration of original stdin/stdout

**From `codec_test.go`:**
```go
func TestDecodeStdin(t *testing.T) {
    // Create temp file with JSON
    f, err := os.CreateTemp("", "codec-test-*.json")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(f.Name())

    want := Payload{Name: "alice", Age: 30}
    json.NewEncoder(f).Encode(want)
    f.Seek(0, 0)

    // Redirect stdin
    orig := os.Stdin
    os.Stdin = f
    defer func() { os.Stdin = orig }()

    // Call function that reads stdin
    var got Payload
    if err := codec.DecodeStdin(&got); err != nil {
        t.Fatalf("DecodeStdin: %v", err)
    }
    
    // Verify
    if got != want {
        t.Fatalf("got %+v, want %+v", got, want)
    }
}
```

## Mocking Patterns

**No external mock framework:**
- Uses Go's capability to reassign package-level variables
- Focuses on testing pure functions and state changes

**Example: Route dispatch testing**
- Sets `os.Args` to simulate command-line invocation
- Registers test route
- Calls `Dispatch()` and verifies handler was invoked
- Cleans up with `ClearRoutes()`

**os.Stdin/Stdout redirection:**
Most thorough mocking approach in codebase
- Swap out standard file descriptors
- Write/read from temp files
- Restore originals in defer

## Assertion Patterns

**Assertion style (no helper library):**
```go
if condition {
    t.Fatal("message")    // Immediate stop, error message
    // or
    t.Fatalf("got %v, want %v", got, want)  // Formatted stop
}
```

**Pointer assertions:**
```go
if r.Proceed == nil || !*r.Proceed {
    t.Fatal("LetStop should set Proceed=true")
}
```

**Struct field assertions:**
```go
if a.Details == nil || a.Details.Decision != "allow" {
    t.Fatal("ApproveToolUse should set decision=allow")
}
```

**String value assertions:**
```go
if b.Decision != "block" {
    t.Fatalf("HaltAndContinue: Decision=%q, want block", b.Decision)
}
```

## What's Tested

**Core routing logic (`agenthooks_test.go`):**
- Route registration and dispatch
- Event processing from stdin/stdout
- Verdict helper functions
- Agent loop detection (IsLooping)
- Unified agent ID constants

**Response helpers (platform-specific tests):**
- `claude/responses_test.go`:
  - StopEvent responses (LetStop, HaltAndContinue)
  - PreToolUse responses (ApproveToolUse, DenyToolUse, AskUserAboutTool)
  - PostToolUse responses (AcknowledgeToolUse, AddToolContext, RejectToolResult)
  - UserPromptSubmit responses (ApprovePrompt, RejectPrompt, AppendToPrompt)

- `cursor/responses_test.go`:
  - Permission helpers (Permit, Forbid, RequestConfirmation)
  - Stop helpers (LetStop, SendFollowup)
  - Prompt helpers (AcceptPrompt, BlockPrompt)

**Codec layer (`internal/codec/codec_test.go`):**
- DecodeStdin with valid JSON
- DecodeStdin with invalid JSON (error handling)
- EncodeStdout and round-trip verification

## What's NOT Tested (Gaps)

**Missing test coverage:**
1. **Windsurf, Droid, Gemini platform helpers** - No tests for `windsurf/responses.go`, `droid/responses.go`, `gemini/responses.go`
2. **Unified event translation** - `unified.go` maps all platform events but has no direct unit tests
3. **Install and build commands** - `cmd/agenthooks/` (install, build, scaffold) not tested
4. **UTF-8 BOM handling** - Codec has special case for Cursor Windows BOM but no explicit test
5. **Error codes and exit behavior** - Tests verify behavior but don't assert exact exit codes

## Testing Best Practices Applied

**What's done well:**
1. **Isolation** - Each test is independent; `ClearRoutes()` resets state
2. **Cleanup** - All temp files have `defer os.Remove()`; all file descriptor redirects use `defer` to restore
3. **Clarity** - Test names describe what's being tested
4. **Data-driven** - Subtable pattern for multiple scenarios
5. **Focused scope** - Each test file tests one concern (routes, codec, or responses)

**What could improve:**
1. **Error message assertions** - Currently just check error presence, not content
2. **Integration tests** - No end-to-end tests verifying multiple handlers together
3. **Platform coverage** - Only Claude and Cursor response helpers tested (missing Windsurf, Droid, Gemini)
4. **Exit code verification** - Tests don't assert on `os.Exit()` codes

## Coverage Targets

**No coverage requirement enforced** - No `.coverprofile` mentioned or coverage threshold in tests

**Estimate from inspection:**
- Core `agenthooks.go` routing: ~80% coverage
- Response helpers (Claude/Cursor): ~100% coverage
- Codec layer: ~90% coverage
- Unified event translation: ~0% direct coverage (only indirect through integration)
- Platform-specific responses (Windsurf/Droid/Gemini): ~0% coverage

## Test Execution and Debugging

**Running tests locally:**
```bash
go test ./...                    # All tests
go test -v ./...                 # Verbose
go test ./agenthooks_test.go     # Specific test file
go test -run TestIsLooping ./    # Specific test by name
```

**Debugging approach:**
- Tests use straightforward assertions
- Failures print actual vs expected values
- `t.Fatalf()` stops at first failure with formatted message
- Use `-v` flag to see each test execution step

## Test Dependencies

**External:**
- No test framework dependencies
- Uses only Go standard library: `testing`, `os`, `encoding/json`

**Internal:**
- Tests import the package being tested
- Cross-package tests (e.g., `agenthooks_test` imports `agenthooks`)

---

*Testing analysis: 2026-04-28*

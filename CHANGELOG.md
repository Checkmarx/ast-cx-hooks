# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.3] - 2026-06-19

### Added
- Extended GitHub Copilot **CLI** support: richer `copilotcli` types, response
  builders, and adapters to match the Copilot CLI hook schema
  (`AST-159430-copilot-schema-changes`, #14).
- Tool-naming conventions expanded in `internal/hookcore` to cover the Copilot CLI
  wire format (lowercase tool names, flat output).
- `install` coverage for the updated Copilot CLI route, with additional installer tests.

### Changed
- CI and release workflows now use concurrency groups to serialize job execution
  (#14).
- Hardened workflow security: tightened `Hooks-Release` permissions (read for
  contents, scoped write on the release job) and disabled credential persistence on
  checkout steps ([StepSecurity], #13).

## [1.0.2] - 2026-06-09

### Added
- **GitHub Copilot (VS Code)** support with unified hooks for `PreToolUse`,
  `PostToolUse`, `UserPromptSubmit`, and `Stop`, plus tests (`AST-157916`, #12).

### Changed
- Major internal refactor to the unified hook architecture: replaced the old codec
  with `internal/hookcore` for stdin/stdout processing, added a registry for unified
  handlers, and introduced `aliases.go` to re-export the unified vocabulary as the
  public `agenthooks.*` API (#12).
- Broadened cross-agent test coverage for unified routes and platform adapters.
- Updated README to document the unified hook structure.

### Removed
- Obsolete internal documentation (architecture, concerns, conventions, integrations,
  stack, structure, testing) now superseded by the refactor (#12).

## [1.0.1] - 2026-06-09

### Added
- `BeforeFileEdit` hook for Claude Code — gate a file write before it lands, with the
  proposed changes available so a handler can deny a vulnerable write (#11).
- Importable `install` package (`install/agents.go`, `command.go`, `hooks.go`,
  `json.go`) so consumers can wire hook configuration into their own CLI (#11).

### Changed
- Simplified `cmd/agenthooks/main.go` by moving installation logic into the `install`
  package (#11).
- Applied security best practices to CI/workflows ([StepSecurity], #7, #9).

## [1.0.0] - 2026-03-27

### Added
- Initial release of the **cxagenthooks** framework — one hook codebase for every AI
  coding agent. A single Go handler compiles to one binary that runs across Claude
  Code, Cursor, Windsurf Cascade, Factory Droid, Gemini CLI, and GitHub Copilot,
  with the library translating each platform's wire format.
- Unified hook categories mapping to each platform's native events, with per-platform
  packages (`claude`, `cursor`, `windsurf`, `droid`, `gemini`) and response builders.
- Named-scenario dispatch so multiple handler implementations can share one hook
  (fail-closed on the gating hooks).
- `cmd/agenthooks` CLI: `init` (scaffold a starter project), `build` (build /
  cross-compile), and `install` (write per-agent hook config).
- Release automation via the `Hooks-Release.yml` workflow, including a version regex
  rule (`AST-143344`, #3).
- Zero runtime dependencies (standard library only).

[Unreleased]: https://github.com/Checkmarx/ast-cx-hooks/compare/v1.0.3...HEAD
[1.0.3]: https://github.com/Checkmarx/ast-cx-hooks/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/Checkmarx/ast-cx-hooks/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/Checkmarx/ast-cx-hooks/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/Checkmarx/ast-cx-hooks/releases/tag/v1.0.0

package claude

import (
	"encoding/json"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// --- Stop responses ---

// LetStop returns a decision that allows Claude to stop normally.
func LetStop() StopResult {
	return StopResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// HaltAndContinue blocks Claude from stopping and provides feedback to continue working.
func HaltAndContinue(reason string) StopResult {
	return StopResult{Decision: "block", Reason: reason}
}

// --- PreToolUse responses ---

// ApproveToolUse allows the tool call with no message.
func ApproveToolUse() PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{EventName: "PreToolUse", Decision: "allow"},
	}
}

// ApproveToolUseWithNote allows the tool call and surfaces a note to the user.
func ApproveToolUseWithNote(note string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "allow", DecisionReason: note,
		},
	}
}

// ApproveToolUseWithInput allows the tool call but rewrites its input before execution.
// updated is the full replacement tool_input as raw JSON.
func ApproveToolUseWithInput(updated json.RawMessage) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{EventName: "PreToolUse", Decision: "allow", RewrittenInput: updated},
	}
}

// ApproveToolUseWithContext allows the tool call and injects additional context for the agent.
func ApproveToolUseWithContext(ctx string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "allow", ExtraContext: ctx,
		},
	}
}

// DenyToolUse blocks the tool call and sends reason to the agent.
func DenyToolUse(reason string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "deny", DecisionReason: reason,
		},
	}
}

// DenyToolUseWithContext blocks the tool call, sends reason to the agent, and
// injects additionalContext alongside the denial (e.g. an instruction to run a
// remediation skill).
func DenyToolUseWithContext(reason, ctx string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "deny", DecisionReason: reason, ExtraContext: ctx,
		},
	}
}

// DeferToolUse defers the permission decision to the next handler or the default flow.
func DeferToolUse(reason string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "defer", DecisionReason: reason,
		},
	}
}

// AskUserAboutTool asks the user to confirm before allowing the tool call.
func AskUserAboutTool(reason string) PreToolUseResult {
	return PreToolUseResult{
		Details: &ToolPermission{
			EventName: "PreToolUse", Decision: "ask", DecisionReason: reason,
		},
	}
}

// --- PostToolUse responses ---

// AcknowledgeToolUse allows normal post-tool flow with no feedback.
func AcknowledgeToolUse() PostToolUseResult {
	return PostToolUseResult{}
}

// AddToolContext appends additional context for the agent after the tool completes.
func AddToolContext(ctx string) PostToolUseResult {
	return PostToolUseResult{
		Details: &PostToolDetails{EventName: "PostToolUse", ExtraContext: ctx},
	}
}

// RejectToolResult injects feedback into the agent after the tool completes.
func RejectToolResult(reason string) PostToolUseResult {
	return PostToolUseResult{Decision: "block", Reason: reason}
}

// RejectToolResultWithContext injects blocking feedback into the agent after the
// tool completes AND appends additionalContext (e.g. an instruction to run a
// remediation skill on the findings that caused the reject). The top-level
// decision/reason carries the block; hookSpecificOutput.additionalContext carries
// the context.
func RejectToolResultWithContext(reason, ctx string) PostToolUseResult {
	return PostToolUseResult{
		Decision: "block",
		Reason:   reason,
		Details:  &PostToolDetails{EventName: "PostToolUse", ExtraContext: ctx},
	}
}

// ReplaceToolOutput replaces the tool's result before the agent sees it.
// Unlike AddToolContext (which appends), this substitutes the output entirely.
func ReplaceToolOutput(updated json.RawMessage) PostToolUseResult {
	return PostToolUseResult{
		Details: &PostToolDetails{EventName: "PostToolUse", UpdatedToolOutput: updated},
	}
}

// --- UserPromptSubmit responses ---

// ApprovePrompt allows the prompt to proceed.
func ApprovePrompt() UserPromptSubmitResult {
	return UserPromptSubmitResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// RejectPrompt blocks the prompt and shows reason to the user.
func RejectPrompt(reason string) UserPromptSubmitResult {
	return UserPromptSubmitResult{Decision: "block", Reason: reason}
}

// AppendToPrompt allows the prompt and injects additional context into the agent's system prompt.
func AppendToPrompt(ctx string) UserPromptSubmitResult {
	return UserPromptSubmitResult{
		ResultBase: ResultBase{Proceed: hookcore.Ptr(true)},
		Details:    &PromptSubmitDetails{EventName: "UserPromptSubmit", ExtraContext: ctx},
	}
}

// SetPromptSessionTitle allows the prompt and sets the session display title.
func SetPromptSessionTitle(title string) UserPromptSubmitResult {
	return UserPromptSubmitResult{
		ResultBase: ResultBase{Proceed: hookcore.Ptr(true)},
		Details:    &PromptSubmitDetails{EventName: "UserPromptSubmit", SessionTitle: title},
	}
}

// RejectAndSuppressPrompt blocks the prompt and hides the original prompt from the transcript.
func RejectAndSuppressPrompt(reason string) UserPromptSubmitResult {
	return UserPromptSubmitResult{
		Decision: "block",
		Reason:   reason,
		Details:  &PromptSubmitDetails{EventName: "UserPromptSubmit", SuppressOriginalPrompt: true},
	}
}

// --- SessionStart responses ---

// AcknowledgeSession allows the session to start with no additional context.
func AcknowledgeSession() SessionStartResult {
	return SessionStartResult{}
}

// InjectSessionContext injects additional context into the agent at session start.
func InjectSessionContext(ctx string) SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{EventName: "SessionStart", ExtraContext: ctx},
	}
}

// SetSessionTitle sets a display title for the session.
func SetSessionTitle(title string) SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{EventName: "SessionStart", SessionTitle: title},
	}
}

// SeedInitialMessage seeds an initial user message into the session.
func SeedInitialMessage(msg string) SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{EventName: "SessionStart", InitialUserMessage: msg},
	}
}

// WatchFiles registers filesystem paths the session should watch for changes.
func WatchFiles(paths ...string) SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{EventName: "SessionStart", WatchPaths: paths},
	}
}

// ReloadSkills requests the agent reload its skill set at session start.
func ReloadSkills() SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{EventName: "SessionStart", ReloadSkills: true},
	}
}

// --- PostCompact responses ---

// AcknowledgeCompact allows the session to continue after compaction with no changes.
func AcknowledgeCompact() PostCompactResult {
	return PostCompactResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// --- SubagentStart responses ---

// AcknowledgeSubagentStart allows the subagent to start with no additional context.
func AcknowledgeSubagentStart() SubagentStartResult {
	return SubagentStartResult{}
}

// InjectSubagentContext injects additional context into the subagent at start.
func InjectSubagentContext(ctx string) SubagentStartResult {
	return SubagentStartResult{
		Details: &SubagentStartDetails{EventName: "SubagentStart", ExtraContext: ctx},
	}
}

// --- SubagentStop responses ---

// LetSubagentStop allows the subagent to stop normally.
func LetSubagentStop() SubagentStopResult {
	return SubagentStopResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// HaltSubagent blocks the subagent from stopping and provides feedback to continue working.
func HaltSubagent(reason string) SubagentStopResult {
	return SubagentStopResult{Decision: "block", Reason: reason}
}

// --- PostToolUseFailure responses ---

// AcknowledgeToolFailure allows normal flow after a tool failure with no feedback.
func AcknowledgeToolFailure() PostToolUseFailureResult {
	return PostToolUseFailureResult{}
}

// AnnotateToolFailure appends additional context for the agent after a tool failure.
func AnnotateToolFailure(ctx string) PostToolUseFailureResult {
	return PostToolUseFailureResult{
		Details: &PostToolUseFailureDetails{EventName: "PostToolUseFailure", ExtraContext: ctx},
	}
}

// RejectAfterFailure surfaces feedback to the agent after a tool failure.
// PostToolUseFailure cannot block, so the reason is delivered as additionalContext.
func RejectAfterFailure(reason string) PostToolUseFailureResult {
	return PostToolUseFailureResult{
		Details: &PostToolUseFailureDetails{EventName: "PostToolUseFailure", ExtraContext: reason},
	}
}

// --- PermissionRequest responses ---

// AllowPermission grants the requested permission, optionally rewriting the tool input.
// Pass a nil updated to allow without modifying the input.
func AllowPermission(updated json.RawMessage) PermissionRequestResult {
	return PermissionRequestResult{
		Details: &PermissionRequestDetails{
			EventName: "PermissionRequest",
			Decision:  &PermissionDecision{Behavior: "allow", UpdatedInput: updated},
		},
	}
}

// DenyPermission denies the requested permission.
func DenyPermission() PermissionRequestResult {
	return PermissionRequestResult{
		Details: &PermissionRequestDetails{
			EventName: "PermissionRequest",
			Decision:  &PermissionDecision{Behavior: "deny"},
		},
	}
}

// --- Setup responses ---

// AcknowledgeSetup allows setup to proceed with no additional context.
func AcknowledgeSetup() SetupResult {
	return SetupResult{}
}

// InjectSetupContext injects additional context into the agent during setup.
func InjectSetupContext(ctx string) SetupResult {
	return SetupResult{
		Details: &SetupDetails{EventName: "Setup", ExtraContext: ctx},
	}
}

// --- InstructionsLoaded responses ---

// AcknowledgeInstructions acknowledges loaded instructions (observational; no decision).
func AcknowledgeInstructions() InstructionsLoadedResult {
	return InstructionsLoadedResult{}
}

// --- PermissionDenied responses ---

// AcceptDenial accepts the permission denial without requesting a retry.
func AcceptDenial() PermissionDeniedResult {
	return PermissionDeniedResult{
		Details: &PermissionDeniedDetails{EventName: "PermissionDenied"},
	}
}

// RequestRetry asks the agent to re-attempt the denied action.
func RequestRetry() PermissionDeniedResult {
	return PermissionDeniedResult{
		Details: &PermissionDeniedDetails{EventName: "PermissionDenied", Retry: true},
	}
}

// --- ConfigChange responses ---

// AllowConfigChange allows the configuration change to proceed.
func AllowConfigChange() ConfigChangeResult {
	return ConfigChangeResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// BlockConfigChange blocks the configuration change and shows reason to the user.
func BlockConfigChange(reason string) ConfigChangeResult {
	return ConfigChangeResult{Decision: "block", Reason: reason}
}

// --- CwdChanged responses ---

// AcknowledgeCwdChange acknowledges the working-directory change (observational).
func AcknowledgeCwdChange() CwdChangedResult {
	return CwdChangedResult{}
}

// --- FileChanged responses ---

// AcknowledgeFileChange acknowledges the file change (observational).
func AcknowledgeFileChange() FileChangedResult {
	return FileChangedResult{}
}

// --- StopFailure responses ---

// AcknowledgeStopFailure acknowledges the stop failure (observational).
func AcknowledgeStopFailure() StopFailureResult {
	return StopFailureResult{}
}

// --- UserPromptExpansion responses ---

// AllowExpansion allows the prompt expansion to proceed.
func AllowExpansion() UserPromptExpansionResult {
	return UserPromptExpansionResult{ResultBase: ResultBase{Proceed: hookcore.Ptr(true)}}
}

// BlockExpansion blocks the prompt expansion and shows reason to the user.
func BlockExpansion(reason string) UserPromptExpansionResult {
	return UserPromptExpansionResult{Decision: "block", Reason: reason}
}

// AnnotateExpansion allows the expansion and injects additional context for the agent.
func AnnotateExpansion(ctx string) UserPromptExpansionResult {
	return UserPromptExpansionResult{
		ResultBase: ResultBase{Proceed: hookcore.Ptr(true)},
		Details:    &UserPromptExpansionDetails{EventName: "UserPromptExpansion", ExtraContext: ctx},
	}
}

// --- WorktreeCreate responses ---

// AcknowledgeWorktreeCreate acknowledges the worktree creation with no override.
func AcknowledgeWorktreeCreate() WorktreeCreateResult {
	return WorktreeCreateResult{}
}

// SetWorktreePath acknowledges the worktree creation and overrides its path.
func SetWorktreePath(path string) WorktreeCreateResult {
	return WorktreeCreateResult{
		Details: &WorktreeCreateDetails{EventName: "WorktreeCreate", WorktreePath: path},
	}
}

// --- WorktreeRemove responses ---

// AcknowledgeWorktreeRemove acknowledges the worktree removal (observational).
func AcknowledgeWorktreeRemove() WorktreeRemoveResult {
	return WorktreeRemoveResult{}
}

// --- Elicitation responses ---

// AcceptElicitation accepts the elicitation request with the provided content.
func AcceptElicitation(content json.RawMessage) ElicitationResult {
	return ElicitationResult{
		Details: &ElicitationDetails{EventName: "Elicitation", Action: "accept", Content: content},
	}
}

// DeclineElicitation declines the elicitation request.
func DeclineElicitation() ElicitationResult {
	return ElicitationResult{
		Details: &ElicitationDetails{EventName: "Elicitation", Action: "decline"},
	}
}

// CancelElicitation cancels the elicitation request.
func CancelElicitation() ElicitationResult {
	return ElicitationResult{
		Details: &ElicitationDetails{EventName: "Elicitation", Action: "cancel"},
	}
}

// --- ElicitationResult responses ---

// AcceptElicitationResult accepts the post-elicitation result with the provided content.
func AcceptElicitationResult(content json.RawMessage) ElicitationResultResult {
	return ElicitationResultResult{
		Details: &ElicitationDetails{EventName: "ElicitationResult", Action: "accept", Content: content},
	}
}

// DeclineElicitationResult declines the post-elicitation result.
func DeclineElicitationResult() ElicitationResultResult {
	return ElicitationResultResult{
		Details: &ElicitationDetails{EventName: "ElicitationResult", Action: "decline"},
	}
}

// CancelElicitationResult cancels the post-elicitation result.
func CancelElicitationResult() ElicitationResultResult {
	return ElicitationResultResult{
		Details: &ElicitationDetails{EventName: "ElicitationResult", Action: "cancel"},
	}
}

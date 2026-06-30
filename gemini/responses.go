package gemini

import (
	"encoding/json"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/hookcore"
)

// --- BeforeTool responses ---

// ApproveToolCall allows the tool call to proceed.
func ApproveToolCall() BeforeToolResult {
	return BeforeToolResult{}
}

// DenyToolCall blocks the tool call; reason is fed back to the model.
func DenyToolCall(reason string) BeforeToolResult {
	return BeforeToolResult{ResultBase: ResultBase{Decision: "deny", Reason: reason}}
}

// DenyToolCallWithContext blocks the tool call and delivers remediation context by
// folding it into reason. Gemini's BeforeTool wire format has no additionalContext
// field — only reason is forwarded to the model.
func DenyToolCallWithContext(reason, ctx string) BeforeToolResult {
	return DenyToolCall(mergeReasonContext(reason, ctx))
}

// mergeReasonContext joins a deny reason with remediation context when the hook
// protocol has no additionalContext channel on BeforeTool.
func mergeReasonContext(reason, ctx string) string {
	if ctx == "" {
		return reason
	}
	if reason == "" {
		return ctx
	}
	return reason + "\n\n" + ctx
}

// ApproveToolCallWithInput allows the tool call but rewrites its input before execution.
// Gemini merges the provided tool_input with the model's arguments.
func ApproveToolCallWithInput(updated json.RawMessage) BeforeToolResult {
	return BeforeToolResult{Details: &BeforeToolDetails{HookEventName: "BeforeTool", RewrittenInput: updated}}
}

// --- AfterTool responses ---

// AcknowledgeToolCall accepts the tool result with no modification.
func AcknowledgeToolCall() AfterToolResult {
	return AfterToolResult{}
}

// AddToolAnnotation appends extra context to the tool result seen by the model.
func AddToolAnnotation(ctx string) AfterToolResult {
	return AfterToolResult{
		Details: &AfterToolDetails{HookEventName: "AfterTool", ExtraContext: ctx},
	}
}

// DenyToolResult hides the real tool output from the model and replaces it with reason.
func DenyToolResult(reason string) AfterToolResult {
	return AfterToolResult{ResultBase: ResultBase{Decision: "deny", Reason: reason}}
}

// DenyToolResultWithContext blocks the tool result (decision="deny" + reason) AND
// appends additionalContext for the model (e.g. an instruction to run a remediation
// skill on the findings that caused the reject). The top-level decision/reason
// carries the block; hookSpecificOutput.additionalContext carries the context.
func DenyToolResultWithContext(reason, ctx string) AfterToolResult {
	return AfterToolResult{
		ResultBase: ResultBase{Decision: "deny", Reason: reason},
		Details:    &AfterToolDetails{HookEventName: "AfterTool", ExtraContext: ctx},
	}
}

// ChainToolCall requests a follow-up tool call whose result replaces the original
// tool response (Gemini's tailToolCallRequest).
func ChainToolCall(name string, args json.RawMessage) AfterToolResult {
	return AfterToolResult{
		Details: &AfterToolDetails{HookEventName: "AfterTool", TailToolCall: &TailToolCall{Name: name, Args: args}},
	}
}

// --- BeforeAgent responses ---

// AcceptTurn allows the agent turn to proceed.
func AcceptTurn() BeforeAgentResult {
	return BeforeAgentResult{}
}

// RejectTurn blocks the agent turn; reason is shown to the user.
func RejectTurn(reason string) BeforeAgentResult {
	return BeforeAgentResult{
		ResultBase: ResultBase{Proceed: hookcore.Ptr(false), Reason: reason},
	}
}

// EnrichTurn allows the turn and appends additional context to the prompt.
func EnrichTurn(ctx string) BeforeAgentResult {
	return BeforeAgentResult{
		Details: &BeforeAgentDetails{HookEventName: "BeforeAgent", ExtraContext: ctx},
	}
}

// DiscardTurn blocks the agent turn AND discards the user message from context.
// Gemini uses decision="deny" for this; unlike RejectTurn (continue=false), which
// blocks the turn but preserves the user message.
func DiscardTurn(reason string) BeforeAgentResult {
	return BeforeAgentResult{
		ResultBase: ResultBase{Decision: "deny", Reason: reason},
	}
}

// --- AfterAgent responses ---

// AcceptResponse accepts the agent's response and ends the turn.
func AcceptResponse() AfterAgentResult {
	return AfterAgentResult{}
}

// RetryWithFeedback rejects the response and triggers a retry with reason as the new prompt.
// Gemini triggers a retry via decision="deny"; continue=false would instead stop the session.
func RetryWithFeedback(reason string) AfterAgentResult {
	return AfterAgentResult{
		ResultBase: ResultBase{Decision: "deny", Reason: reason},
	}
}

// ClearAgentContext accepts the response and clears the model's conversation memory.
func ClearAgentContext() AfterAgentResult {
	return AfterAgentResult{
		Details: &AfterAgentDetails{HookEventName: "AfterAgent", ClearContext: true},
	}
}

// --- BeforeModel responses ---

// OverrideModelRequest replaces the outgoing LLM request before it is sent.
func OverrideModelRequest(req LLMRequest) BeforeModelResult {
	return BeforeModelResult{
		Details: &BeforeModelDetails{HookEventName: "BeforeModel", OverrideRequest: &req},
	}
}

// SyntheticModelResponse supplies a mock LLM response, skipping the actual LLM call.
func SyntheticModelResponse(resp LLMResponse) BeforeModelResult {
	return BeforeModelResult{
		Details: &BeforeModelDetails{HookEventName: "BeforeModel", SyntheticResponse: &resp},
	}
}

// --- AfterModel responses ---

// OverrideModelResponse replaces the received LLM response chunk.
func OverrideModelResponse(resp LLMResponse) AfterModelResult {
	return AfterModelResult{
		Details: &AfterModelDetails{HookEventName: "AfterModel", OverrideResponse: &resp},
	}
}

// --- BeforeToolSelection responses ---

// SetToolConfig constrains which tools the model may select.
func SetToolConfig(cfg ToolConfig) BeforeToolSelectionResult {
	return BeforeToolSelectionResult{
		Details: &ToolSelectionDetails{HookEventName: "BeforeToolSelection", ToolConfig: &cfg},
	}
}

// --- Universal output helpers ---
//
// These build a bare ResultBase that callers compose into a result's embedded
// ResultBase field, e.g.:
//
//	r := gemini.DenyToolCall("nope")
//	r.ResultBase = gemini.Silently()
//
// or merge selected fields onto an existing result's ResultBase.

// Silently suppresses the hook's output from the user (suppressOutput).
func Silently() ResultBase {
	return ResultBase{MuteOutput: true}
}

// WithSystemMessage attaches a system message to the result (systemMessage).
func WithSystemMessage(msg string) ResultBase {
	return ResultBase{SystemNote: msg}
}

// --- SessionStart responses ---

// AcknowledgeSession allows the session to proceed.
func AcknowledgeSession() SessionStartResult { return SessionStartResult{} }

// InjectSessionContext injects context at session start.
func InjectSessionContext(ctx string) SessionStartResult {
	return SessionStartResult{
		Details: &SessionStartDetails{HookEventName: "SessionStart", ExtraContext: ctx},
	}
}

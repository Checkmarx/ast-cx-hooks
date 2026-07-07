package gemini_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Checkmarx/ast-cx-hooks/gemini"
)

// TestRetryWithFeedbackUsesDeny guards the retry-semantics fix: Gemini triggers a
// retry via decision="deny"; continue=false would instead STOP the session.
func TestRetryWithFeedbackUsesDeny(t *testing.T) {
	r := gemini.RetryWithFeedback("try again")
	if r.Decision != "deny" {
		t.Fatalf("RetryWithFeedback: Decision=%q, want deny", r.Decision)
	}
	if r.Reason != "try again" {
		t.Fatalf("RetryWithFeedback: Reason=%q", r.Reason)
	}
	if r.Proceed != nil {
		t.Fatal("RetryWithFeedback must not set continue (that would stop, not retry)")
	}
	b, _ := json.Marshal(r)
	if out := string(b); !strings.Contains(out, `"decision":"deny"`) || strings.Contains(out, `"continue"`) {
		t.Fatalf("unexpected JSON: %s", out)
	}
}

// TestAfterToolOutputRewrite covers the deny/replace and tail-call routing builders.
func TestAfterToolOutputRewrite(t *testing.T) {
	d := gemini.DenyToolResult("redacted")
	if d.Decision != "deny" || d.Reason != "redacted" {
		t.Fatalf("DenyToolResult: %+v", d.ResultBase)
	}

	c := gemini.ChainToolCall("read_file", json.RawMessage(`{"path":"/x"}`))
	if c.Details == nil || c.Details.TailToolCall == nil || c.Details.TailToolCall.Name != "read_file" {
		t.Fatalf("ChainToolCall: %+v", c.Details)
	}
	b, _ := json.Marshal(c)
	if out := string(b); !strings.Contains(out, `"tailToolCallRequest"`) || !strings.Contains(out, `"read_file"`) {
		t.Fatalf("tailToolCallRequest not emitted: %s", out)
	}
}

// TestDenyToolResultWithContext verifies the AfterTool reject-with-context builder
// emits BOTH the block (decision="deny" + reason) AND additionalContext, and that a
// round-trip back through AfterToolResult preserves both.
func TestDenyToolResultWithContext(t *testing.T) {
	r := gemini.DenyToolResultWithContext("secrets detected", "run the cx-remediation skill")
	if r.Decision != "deny" || r.Reason != "secrets detected" {
		t.Fatalf("DenyToolResultWithContext base: %+v", r.ResultBase)
	}
	if r.Details == nil || r.Details.ExtraContext != "run the cx-remediation skill" {
		t.Fatalf("DenyToolResultWithContext details: %+v", r.Details)
	}
	b, _ := json.Marshal(r)
	out := string(b)
	if !strings.Contains(out, `"decision":"deny"`) {
		t.Fatalf("decision not emitted: %s", out)
	}
	if !strings.Contains(out, `"reason":"secrets detected"`) {
		t.Fatalf("reason not emitted: %s", out)
	}
	if !strings.Contains(out, `"additionalContext":"run the cx-remediation skill"`) {
		t.Fatalf("additionalContext not emitted: %s", out)
	}

	// Round-trip back to confirm both decision and additionalContext survive.
	var rt gemini.AfterToolResult
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rt.Decision != "deny" || rt.Reason != "secrets detected" {
		t.Fatalf("round-trip base: %+v", rt.ResultBase)
	}
	if rt.Details == nil || rt.Details.ExtraContext != "run the cx-remediation skill" {
		t.Fatalf("round-trip details: %+v", rt.Details)
	}
}

// TestLLMResponseShape verifies the corrected llm_response shape (candidates + usageMetadata)
// rather than the previous (non-existent) top-level content field.
func TestLLMResponseShape(t *testing.T) {
	// In the hook wire format, content.parts is a plain string array.
	raw := `{"candidates":[{"content":{"role":"model","parts":["hi"]},"finishReason":"STOP"}],"usageMetadata":{"totalTokenCount":42}}`
	var r gemini.LLMResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(r.Candidates) != 1 || r.Candidates[0].Content.Role != "model" {
		t.Fatalf("candidates not parsed: %+v", r.Candidates)
	}
	if len(r.Candidates[0].Content.Parts) != 1 || r.Candidates[0].Content.Parts[0] != "hi" {
		t.Fatalf("parts: %+v", r.Candidates[0].Content.Parts)
	}
	if r.Candidates[0].FinishReason != "STOP" {
		t.Fatalf("finishReason: %q", r.Candidates[0].FinishReason)
	}
	if r.UsageMetadata == nil || r.UsageMetadata.TotalTokenCount != 42 {
		t.Fatalf("usageMetadata: %+v", r.UsageMetadata)
	}
}

// TestLLMRequestToolConfig verifies messages typing and the toolConfig input field.
func TestLLMRequestToolConfig(t *testing.T) {
	raw := `{"model":"gemini-2.0","messages":[{"role":"user","content":"hi"}],"toolConfig":{"mode":"ANY","allowedFunctionNames":["read_file"]}}`
	var rq gemini.LLMRequest
	if err := json.Unmarshal([]byte(raw), &rq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rq.Messages) != 1 || rq.Messages[0].Role != "user" || rq.Messages[0].Content != "hi" {
		t.Fatalf("messages: %+v", rq.Messages)
	}
	if rq.ToolConfig == nil || rq.ToolConfig.Mode != "ANY" {
		t.Fatalf("toolConfig: %+v", rq.ToolConfig)
	}
}

// TestOverrideModelRequest verifies the BeforeModel override emits llm_request.
func TestOverrideModelRequest(t *testing.T) {
	r := gemini.OverrideModelRequest(gemini.LLMRequest{Model: "gemini-2.0"})
	if r.Details == nil || r.Details.OverrideRequest == nil || r.Details.OverrideRequest.Model != "gemini-2.0" {
		t.Fatalf("OverrideModelRequest: %+v", r.Details)
	}
	b, _ := json.Marshal(r)
	if out := string(b); !strings.Contains(out, `"llm_request"`) {
		t.Fatalf("llm_request not emitted: %s", out)
	}
}

// TestSyntheticModelResponse verifies the BeforeModel synthetic response emits llm_response.
func TestSyntheticModelResponse(t *testing.T) {
	r := gemini.SyntheticModelResponse(gemini.LLMResponse{
		Candidates: []gemini.LLMCandidate{{FinishReason: "STOP"}},
	})
	if r.Details == nil || r.Details.SyntheticResponse == nil {
		t.Fatalf("SyntheticModelResponse: %+v", r.Details)
	}
	b, _ := json.Marshal(r)
	if out := string(b); !strings.Contains(out, `"llm_response"`) {
		t.Fatalf("llm_response not emitted: %s", out)
	}
}

// TestOverrideModelResponse verifies the AfterModel override emits llm_response.
func TestOverrideModelResponse(t *testing.T) {
	r := gemini.OverrideModelResponse(gemini.LLMResponse{
		Candidates: []gemini.LLMCandidate{{FinishReason: "STOP"}},
	})
	if r.Details == nil || r.Details.OverrideResponse == nil {
		t.Fatalf("OverrideModelResponse: %+v", r.Details)
	}
	b, _ := json.Marshal(r)
	if out := string(b); !strings.Contains(out, `"llm_response"`) {
		t.Fatalf("llm_response not emitted: %s", out)
	}
}

// TestSetToolConfig verifies the BeforeToolSelection builder emits toolConfig.
func TestSetToolConfig(t *testing.T) {
	r := gemini.SetToolConfig(gemini.ToolConfig{Mode: "ANY", AllowedFunctionNames: []string{"read_file"}})
	if r.Details == nil || r.Details.ToolConfig == nil || r.Details.ToolConfig.Mode != "ANY" {
		t.Fatalf("SetToolConfig: %+v", r.Details)
	}
	b, _ := json.Marshal(r)
	if out := string(b); !strings.Contains(out, `"toolConfig"`) || !strings.Contains(out, `"read_file"`) {
		t.Fatalf("toolConfig not emitted: %s", out)
	}
}

// TestClearAgentContext verifies the AfterAgent builder emits clearContext.
func TestClearAgentContext(t *testing.T) {
	r := gemini.ClearAgentContext()
	if r.Details == nil || !r.Details.ClearContext {
		t.Fatalf("ClearAgentContext: %+v", r.Details)
	}
	b, _ := json.Marshal(r)
	if out := string(b); !strings.Contains(out, `"clearContext":true`) {
		t.Fatalf("clearContext not emitted: %s", out)
	}
}

// TestDiscardTurn verifies the discard path uses decision=deny (distinct from RejectTurn).
func TestDiscardTurn(t *testing.T) {
	r := gemini.DiscardTurn("blocked")
	if r.Decision != "deny" || r.Reason != "blocked" {
		t.Fatalf("DiscardTurn: %+v", r.ResultBase)
	}
	if r.Proceed != nil {
		t.Fatal("DiscardTurn must not set continue (deny discards; continue=false preserves)")
	}
	b, _ := json.Marshal(r)
	if out := string(b); !strings.Contains(out, `"decision":"deny"`) || strings.Contains(out, `"continue"`) {
		t.Fatalf("unexpected JSON: %s", out)
	}
}

// TestUniversalOutputHelpers verifies Silently/WithSystemMessage emit their keys.
func TestUniversalOutputHelpers(t *testing.T) {
	s := gemini.Silently()
	if !s.MuteOutput {
		t.Fatalf("Silently: %+v", s)
	}
	b, _ := json.Marshal(s)
	if out := string(b); !strings.Contains(out, `"suppressOutput":true`) {
		t.Fatalf("suppressOutput not emitted: %s", out)
	}

	m := gemini.WithSystemMessage("note")
	if m.SystemNote != "note" {
		t.Fatalf("WithSystemMessage: %+v", m)
	}
	b2, _ := json.Marshal(m)
	if out := string(b2); !strings.Contains(out, `"systemMessage":"note"`) {
		t.Fatalf("systemMessage not emitted: %s", out)
	}
}

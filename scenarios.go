package agenthooks

import (
	"fmt"
	"os"

	"github.com/Checkmarx/ast-cx-hooks/claude"
	"github.com/Checkmarx/ast-cx-hooks/codex"
	"github.com/Checkmarx/ast-cx-hooks/copilot"
	"github.com/Checkmarx/ast-cx-hooks/copilotcli"
	"github.com/Checkmarx/ast-cx-hooks/cursor"
	"github.com/Checkmarx/ast-cx-hooks/droid"
	"github.com/Checkmarx/ast-cx-hooks/gemini"
	"github.com/Checkmarx/ast-cx-hooks/windsurf"
)

// Scenarios let several independent implementations share one hook. Each is
// registered under a name (e.g. team "phoenix"); the active scenario key is
// chosen per invocation by the selector (default: the <scenario> CLI arg or the
// AGENTHOOKS_SCENARIO env var). When no key matches, the default handler runs;
// when there is no default either, a fallback verdict is returned — restrictive
// (fail closed) for the gating hooks (tool call, file read, prompt) so a missing
// handler never silently permits an action, permissive for the observational ones.

// activeSelector decides the scenario key for every hook. Override with
// UseScenarioSelector (e.g. to pick a scenario from the event content).
var activeSelector ScenarioSelector = ScenarioFromArg

// UseScenarioSelector overrides how the active scenario key is chosen for all hooks.
// A nil selector is ignored. The default is ScenarioFromArg.
func UseScenarioSelector(sel ScenarioSelector) {
	if sel != nil {
		activeSelector = sel
	}
}

// scenarioReg holds the default handler plus the named scenario handlers for one
// hook category, and knows how to register that category's platform adapters.
type scenarioReg[E any, V any] struct {
	def       func(E) V
	scenarios map[string]func(E) V
	fallback  V // returned when neither a scenario nor a default matches
	metaOf    func(E) ScenarioMeta
	register  func(func(E) V) // registers the platform adapters with the given dispatch fn
}

func (r *scenarioReg[E, V]) setDefault(fn func(E) V) {
	r.def = fn
	r.register(r.dispatch)
}

func (r *scenarioReg[E, V]) addScenario(name string, fn func(E) V) {
	if r.scenarios == nil {
		r.scenarios = map[string]func(E) V{}
	}
	r.scenarios[name] = fn
	r.register(r.dispatch)
}

func (r *scenarioReg[E, V]) reset() {
	r.def = nil
	r.scenarios = nil
}

// dispatch resolves the active scenario for this invocation and runs the matching
// handler: a named scenario if the key matches, else the default, else the fallback.
func (r *scenarioReg[E, V]) dispatch(e E) V {
	if key := activeSelector(r.metaOf(e)); key != "" {
		if fn, ok := r.scenarios[key]; ok {
			return fn(e)
		}
		fmt.Fprintf(os.Stderr, "agenthooks: scenario %q not registered; falling back to default handler\n", key)
	}
	if r.def != nil {
		return r.def(e)
	}
	// No scenario matched and no default handler is registered — a misconfiguration.
	// Gating hooks (tool call, file read, prompt) use a restrictive fallback so a
	// missing handler fails CLOSED rather than silently permitting the action; the
	// warning is always emitted (including the empty-key case) so it isn't silent.
	fmt.Fprintln(os.Stderr, "agenthooks: no matching scenario and no default handler registered; using fallback verdict")
	return r.fallback
}

// resetScenarios clears all scenario state and restores the default selector.
// Called by ClearRoutes so tests stay isolated.
func resetScenarios() {
	idleReg.reset()
	subagentIdleReg.reset()
	toolCallReg.reset()
	toolFailureReg.reset()
	fileWriteReg.reset()
	fileEditReg.reset()
	fileReadReg.reset()
	promptReg.reset()
	activeSelector = ScenarioFromArg
}

// --- one registry per hook category ---

var idleReg = &scenarioReg[AgentIdleEvent, IdleVerdict]{
	fallback: Resume(),
	metaOf: func(e AgentIdleEvent) ScenarioMeta {
		return ScenarioMeta{Agent: e.Agent, WorkDir: e.WorkDir, Raw: e.Raw}
	},
	register: func(d func(AgentIdleEvent) IdleVerdict) {
		registerAdapters(AgentIdleFunc(d),
			claude.IdleAdapter, cursor.IdleAdapter, windsurf.IdleAdapter,
			droid.IdleAdapter, gemini.IdleAdapter, copilot.IdleAdapter, copilotcli.IdleAdapter,
			codex.IdleAdapter)
	},
}

var subagentIdleReg = &scenarioReg[AgentIdleEvent, IdleVerdict]{
	fallback: Resume(),
	metaOf: func(e AgentIdleEvent) ScenarioMeta {
		return ScenarioMeta{Agent: e.Agent, WorkDir: e.WorkDir, Raw: e.Raw}
	},
	register: func(d func(AgentIdleEvent) IdleVerdict) {
		registerAdapters(AgentIdleFunc(d),
			claude.SubagentIdleAdapter, droid.SubagentIdleAdapter,
			copilot.SubagentIdleAdapter, cursor.SubagentIdleAdapter, copilotcli.SubagentIdleAdapter,
			codex.SubagentIdleAdapter)
	},
}

var toolCallReg = &scenarioReg[ToolCallEvent, ToolVerdict]{
	fallback: Deny("agenthooks: no handler registered for this tool call (failing closed)"), // gate → fail closed
	metaOf: func(e ToolCallEvent) ScenarioMeta {
		return ScenarioMeta{Agent: e.Agent, WorkDir: e.WorkDir, Raw: e.Raw}
	},
	register: func(d func(ToolCallEvent) ToolVerdict) {
		registerAdapters(ToolCallFunc(d),
			claude.ToolAdapter, cursor.ShellToolAdapter, cursor.MCPToolAdapter,
			windsurf.RunCommandAdapter, windsurf.MCPToolAdapter,
			droid.ToolAdapter, gemini.ToolAdapter, copilot.ToolAdapter, copilotcli.ToolAdapter,
			codex.ToolAdapter)
	},
}

var toolFailureReg = &scenarioReg[ToolFailureEvent, ToolFailureVerdict]{
	fallback: AcknowledgeFailure(),
	metaOf: func(e ToolFailureEvent) ScenarioMeta {
		return ScenarioMeta{Agent: e.Agent, WorkDir: e.WorkDir, Raw: e.Raw}
	},
	register: func(d func(ToolFailureEvent) ToolFailureVerdict) {
		registerAdapters(ToolFailureFunc(d),
			claude.ToolFailureAdapter, cursor.ToolFailureAdapter, copilotcli.ToolFailureAdapter)
	},
}

var fileWriteReg = &scenarioReg[FileWriteEvent, FileWriteVerdict]{
	fallback: AcceptWrite(),
	metaOf: func(e FileWriteEvent) ScenarioMeta {
		return ScenarioMeta{Agent: e.Agent, WorkDir: e.WorkDir, Raw: e.Raw}
	},
	register: func(d func(FileWriteEvent) FileWriteVerdict) {
		registerAdapters(FileWriteFunc(d),
			claude.FileWriteAdapter, cursor.FileWriteAdapter, windsurf.FileWriteAdapter,
			droid.FileWriteAdapter, gemini.FileWriteAdapter, copilot.FileWriteAdapter, copilotcli.FileWriteAdapter,
			codex.FileWriteAdapter)
	},
}

var fileEditReg = &scenarioReg[FileEditEvent, FileEditVerdict]{
	fallback: RejectEdit("agenthooks: no handler registered for this file edit (failing closed)"), // gate → fail closed
	metaOf: func(e FileEditEvent) ScenarioMeta {
		return ScenarioMeta{Agent: e.Agent, WorkDir: e.WorkDir, Raw: e.Raw}
	},
	register: func(d func(FileEditEvent) FileEditVerdict) {
		// Every platform with a pre-execution gate that can block a write before it
		// lands: Claude/Droid/Copilot/Copilot-CLI PreToolUse (Write/Edit tools),
		// Gemini BeforeTool (write_file/replace), Windsurf pre_write_code (exit-2
		// block), and Cursor's preToolUse Write gate. cursor.FileReadAsEditAdapter
		// also routes Cursor's beforeReadFile through this handler (read modelled as a
		// file event with empty Changes) so a single handler gates both, matching
		// consumers that register only BeforeFileEdit.
		registerAdapters(FileEditFunc(d),
			claude.FileEditAdapter, droid.FileEditAdapter, gemini.FileEditAdapter,
			copilot.FileEditAdapter, copilotcli.FileEditAdapter,
			windsurf.FileEditAdapter, cursor.FileEditAdapter, cursor.FileReadAsEditAdapter,
			codex.FileEditAdapter)
	},
}

var fileReadReg = &scenarioReg[FileReadEvent, FileReadVerdict]{
	fallback: DenyRead("agenthooks: no handler registered for this file read (failing closed)"), // gate → fail closed
	metaOf: func(e FileReadEvent) ScenarioMeta {
		return ScenarioMeta{Agent: e.Agent, WorkDir: e.WorkDir, Raw: e.Raw}
	},
	register: func(d func(FileReadEvent) FileReadVerdict) {
		registerAdapters(FileReadFunc(d), cursor.FileReadAdapter, windsurf.FileReadAdapter)
	},
}

var promptReg = &scenarioReg[PromptEvent, PromptVerdict]{
	fallback: RejectPrompt("agenthooks: no handler registered for this prompt (failing closed)"), // gate → fail closed
	metaOf:   func(e PromptEvent) ScenarioMeta { return ScenarioMeta{Agent: e.Agent, Raw: e.Raw} },
	register: func(d func(PromptEvent) PromptVerdict) {
		registerAdapters(PromptFunc(d),
			claude.PromptAdapter, cursor.PromptAdapter, windsurf.PromptAdapter,
			droid.PromptAdapter, gemini.PromptAdapter, copilot.PromptAdapter, copilotcli.PromptAdapter,
			codex.PromptAdapter)
	},
}

// --- named-scenario registration (one per unified hook) ---

// WhenAgentIdleScenario registers a named scenario handler for the agent-idle hook.
func WhenAgentIdleScenario(name string, fn AgentIdleFunc) { idleReg.addScenario(name, fn) }

// WhenSubagentIdleScenario registers a named scenario handler for the subagent-idle hook.
func WhenSubagentIdleScenario(name string, fn AgentIdleFunc) { subagentIdleReg.addScenario(name, fn) }

// BeforeToolCallScenario registers a named scenario handler for the pre-tool-call hook.
func BeforeToolCallScenario(name string, fn ToolCallFunc) { toolCallReg.addScenario(name, fn) }

// AfterToolFailureScenario registers a named scenario handler for the tool-failure hook.
func AfterToolFailureScenario(name string, fn ToolFailureFunc) { toolFailureReg.addScenario(name, fn) }

// AfterFileWriteScenario registers a named scenario handler for the post-file-write hook.
func AfterFileWriteScenario(name string, fn FileWriteFunc) { fileWriteReg.addScenario(name, fn) }

// BeforeFileEditScenario registers a named scenario handler for the pre-file-write hook.
func BeforeFileEditScenario(name string, fn FileEditFunc) { fileEditReg.addScenario(name, fn) }

// BeforeFileReadScenario registers a named scenario handler for the pre-file-read hook.
func BeforeFileReadScenario(name string, fn FileReadFunc) { fileReadReg.addScenario(name, fn) }

// BeforePromptScenario registers a named scenario handler for the prompt-submit hook.
func BeforePromptScenario(name string, fn PromptFunc) { promptReg.addScenario(name, fn) }

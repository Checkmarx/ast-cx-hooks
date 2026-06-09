package hookcore

import "os"

// ScenarioMeta is the context handed to a ScenarioSelector when choosing which
// named scenario handler should run for a hook invocation.
type ScenarioMeta struct {
	Agent   AgentID // which agent triggered the hook
	WorkDir string  // working directory, when the event carries one
	Raw     any     // the platform-specific event, for deep content-based selection
}

// ScenarioSelector returns the active scenario key for an invocation, or "" to
// fall back to the default handler.
type ScenarioSelector func(ScenarioMeta) string

// ScenarioFromArg is the default selector. It reads the scenario key the caller
// passes on the command line as the third argument — `myhook <route> <scenario>`
// — falling back to the AGENTHOOKS_SCENARIO environment variable.
func ScenarioFromArg(ScenarioMeta) string {
	if len(os.Args) > 2 && os.Args[2] != "" {
		return os.Args[2]
	}
	return os.Getenv("AGENTHOOKS_SCENARIO")
}

// ScenarioFromEnv selects the scenario solely from the AGENTHOOKS_SCENARIO
// environment variable.
func ScenarioFromEnv(ScenarioMeta) string {
	return os.Getenv("AGENTHOOKS_SCENARIO")
}

package render

import (
	"fmt"
	"strings"
)

// AgentView is a read-only view of an agent step for vocabulary phrasing.
type AgentView struct {
	Name         string
	Decl         string
	ControlLabel string
	In           string
	PrevProducer string
	UsesFlowArg  bool
}

// GateView is a read-only view of a gate step for vocabulary phrasing.
type GateView struct {
	Name         string
	Run          string
	OnFail       string
	OnFailTarget string
	ScriptRetry  int
}

// StepView is a read-only view of a parallel branch step.
type StepView struct {
	ControlLabel string
	Decl         string
	Kind         string
}

// Vocabulary supplies host-specific verb phrases. Render never hardcodes host nouns;
// bindings (M7/M10) implement this interface.
type Vocabulary interface {
	InvokeAgent(a AgentView) string
	RunScript(g GateView) string
	SpawnParallel(branches []StepView) string
	ReadOutput(valueLabel string) string
	ParseOutputProtocol(enum []string, retry int) string
	GotoStep(controlLabel string) string
	Arg(name string) string
}

type defaultVocabulary struct{}

// DefaultVocabulary returns the host-neutral vocabulary used by M6 tests.
func DefaultVocabulary() Vocabulary {
	return defaultVocabulary{}
}

func (defaultVocabulary) InvokeAgent(a AgentView) string {
	s := fmt.Sprintf("Invoke the `%s` subagent", a.Decl)
	if a.UsesFlowArg {
		s += fmt.Sprintf(" with %s", defaultVocabulary{}.Arg("input"))
	}
	if a.PrevProducer != "" {
		s += fmt.Sprintf(" using the output from `%s`", a.PrevProducer)
	}
	return s + "."
}

func (defaultVocabulary) RunScript(g GateView) string {
	return fmt.Sprintf("Run `%s`.", g.Run)
}

func (defaultVocabulary) SpawnParallel(branches []StepView) string {
	names := make([]string, len(branches))
	for i, b := range branches {
		names[i] = b.Decl
	}
	return fmt.Sprintf("Spawn these subagents together: %s.", strings.Join(names, ", "))
}

func (defaultVocabulary) ReadOutput(valueLabel string) string {
	return fmt.Sprintf("Read the `out:` value from `%s`.", valueLabel)
}

func (defaultVocabulary) ParseOutputProtocol(enum []string, retry int) string {
	return fmt.Sprintf(
		"Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `%d` times, then stop the flow. Allowed values: %s.",
		retry,
		strings.Join(enum, ", "),
	)
}

func (defaultVocabulary) GotoStep(controlLabel string) string {
	return fmt.Sprintf("Go back to step `%s`.", controlLabel)
}

func (defaultVocabulary) Arg(name string) string {
	return fmt.Sprintf("<%s>", name)
}

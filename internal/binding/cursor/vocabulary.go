package cursor

import (
	"fmt"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/render"
)

type cursorVocabulary struct{}

// Vocabulary returns the Cursor render.Vocabulary implementation.
func Vocabulary() render.Vocabulary {
	return cursorVocabulary{}
}

func (cursorVocabulary) InvokeAgent(a render.AgentView) string {
	var s string
	if a.ControlLabel != "" && a.ControlLabel != a.Decl {
		s = fmt.Sprintf(
			"Use the `%s` subagent (`/%s`) (step `%s`)",
			a.Decl, a.Decl, a.ControlLabel,
		)
	} else {
		s = fmt.Sprintf("Use the `%s` subagent (`/%s`)", a.Decl, a.Decl)
	}
	if a.UsesFlowArg {
		s += fmt.Sprintf(" with %s", flowInputArg())
	}
	if a.PrevProducer != "" {
		s += fmt.Sprintf(" using the output from `%s`", a.PrevProducer)
	}
	return s + "."
}

func (cursorVocabulary) RunScript(g render.GateView) string {
	return fmt.Sprintf("Run `%s` in the terminal.", g.Run)
}

func (cursorVocabulary) SpawnParallel(branches []render.StepView) string {
	names := make([]string, len(branches))
	for i, b := range branches {
		names[i] = b.Decl
	}
	return fmt.Sprintf(
		"Launch the following subagents in parallel using multiple Task calls in one message: %s.",
		strings.Join(names, ", "),
	)
}

func (cursorVocabulary) ReadOutput(valueLabel string) string {
	return fmt.Sprintf("Read the `out:` value from `%s`.", valueLabel)
}

func (cursorVocabulary) ParseOutputProtocol(enum []string, retry int) string {
	return fmt.Sprintf(
		"Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `%d` times, then stop the flow. Allowed values: %s.",
		retry,
		strings.Join(enum, ", "),
	)
}

func (cursorVocabulary) GotoStep(controlLabel string) string {
	return fmt.Sprintf("Go back to step `%s`.", controlLabel)
}

// Arg returns the Cursor slash-command positional token for flow input.
// v0.1 maps the opaque entry input to $1 regardless of the IR param name.
func (cursorVocabulary) Arg(_ string) string {
	return flowInputArg()
}

func flowInputArg() string {
	return "$1"
}

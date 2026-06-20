package render

import "strings"

// OutputProtocol emits the spec §9.1 block appended to enum agent prompts.
// Returns empty string when enum is empty.
func OutputProtocol(enum []string, retry int) string {
	if len(enum) == 0 {
		return ""
	}
	_ = retry // retry is documented in runbook via ParseOutputProtocol, not in prompt block
	members := strings.Join(enum, ", ")
	return "When you have finished, end your reply with exactly one fenced block in this form\n" +
		"(no other text after the closing fence):\n\n" +
		"```agentflow-output\n" +
		"out: <value>\n" +
		"```\n\n" +
		"`<value>` must be exactly one of: " + members + "."
}

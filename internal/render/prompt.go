package render

import (
	"strings"

	"github.com/Abhinand20/agentFlow/internal/ir"
)

// AgentPrompt renders an agent's instruction body from resolved IR prompt text.
// Render never reads the filesystem; file- and inline-sourced prompts are identical.
func AgentPrompt(a ir.Agent) string {
	body := a.Prompt
	if len(a.OutEnum) > 0 {
		proto := OutputProtocol(a.OutEnum, a.Retry)
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		if body != "" {
			body += "\n"
		}
		body += proto
	}
	return body
}

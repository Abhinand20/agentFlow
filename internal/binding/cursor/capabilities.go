package cursor

import (
	"fmt"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/ir"
)

// Capability constants name host features for negotiation (spec §11).
type Capability = binding.Capability

const (
	CapCommandTrigger Capability = "command-trigger"
	CapNamedSubagents Capability = "named-subagents"
	CapMCP            Capability = "mcp"
	CapHooks          Capability = "hooks"
	CapParallelSpawn  Capability = "parallel-spawn"
	CapBlockingGate   Capability = "blocking-gate"
	CapOutputParse    Capability = "output-parse"
	CapLoopEnforce    Capability = "loop-enforcement"
)

// StaticCaveat documents a binding-level limitation not tied to a specific program.
const StaticCaveat = "Cursor agents are emitted as .cursor/rules, not native subagents (see CapNamedSubagents)"

type needs struct {
	parallel    bool
	loopBound   bool
	outputParse bool
	blocking    []string
}

type negotiationRule struct {
	cap     Capability
	need    func(needs) bool
	code    string
	message func(needs) string
}

func negotiationRules() []negotiationRule {
	return []negotiationRule{
		{
			cap:  CapParallelSpawn,
			need: func(n needs) bool { return n.parallel },
			code: "AF300",
			message: func(needs) string {
				return "parallel spawn is sequential on cursor"
			},
		},
		{
			cap:  CapLoopEnforce,
			need: func(n needs) bool { return n.loopBound },
			code: "AF301",
			message: func(needs) string {
				return "loop bound is advisory; host self-counts"
			},
		},
		{
			cap:  CapOutputParse,
			need: func(n needs) bool { return n.outputParse },
			code: "AF302",
			message: func(needs) string {
				return "output parsing is advisory"
			},
		},
	}
}

func (c cursorBinding) Capabilities() map[binding.Capability]bool {
	return map[binding.Capability]bool{
		CapCommandTrigger: true,
		CapNamedSubagents: false,
		CapMCP:            true,
		CapHooks:          false,
		CapParallelSpawn:  false,
		CapBlockingGate:   false,
		CapOutputParse:    false,
		CapLoopEnforce:    false,
	}
}

// Negotiate diffs program needs vs binding capabilities and returns AF3xx warnings.
func Negotiate(p ir.Program, caps map[binding.Capability]bool) diag.Diagnostics {
	return negotiateNeeds(scanNeeds(p), caps)
}

func scanNeeds(p ir.Program) needs {
	var n needs
	for _, a := range p.Agents {
		if len(a.OutEnum) > 0 {
			n.outputParse = true
		}
	}
	for _, g := range p.Gates {
		if g.Behavior == "blocking" {
			n.blocking = append(n.blocking, g.Name)
		}
	}
	scanNode(p.Flow.Root, &n)
	return n
}

func scanNode(n ir.Node, needs *needs) {
	switch n.Kind {
	case ir.NodeSeq:
		for _, child := range n.Steps {
			scanNode(child, needs)
		}
	case ir.NodeStep:
		if n.Step != nil && len(n.Step.OutEnum) > 0 {
			needs.outputParse = true
		}
	case ir.NodeBranch:
		for _, c := range n.Cases {
			scanNode(c.Body, needs)
		}
	case ir.NodeLoop:
		if n.HasMax {
			needs.loopBound = true
		}
		if n.Body != nil {
			scanNode(*n.Body, needs)
		}
	case ir.NodeParallel:
		needs.parallel = true
		for _, b := range n.Branches {
			scanNode(b, needs)
		}
		if n.Gather != nil && len(n.Gather.OutEnum) > 0 {
			needs.outputParse = true
		}
	}
}

func negotiateNeeds(n needs, caps map[binding.Capability]bool) diag.Diagnostics {
	var out diag.Diagnostics
	for _, rule := range negotiationRules() {
		if caps[rule.cap] || !rule.need(n) {
			continue
		}
		out.Add(warn(rule.code, rule.message(n)))
	}
	if !caps[CapBlockingGate] {
		for _, name := range n.blocking {
			out.Add(warn("AF303", fmt.Sprintf("gate %q falls back to advisory; cursor hooks deferred", name)))
		}
	}
	return out
}

func agentMappingDiags(agent ir.Agent) diag.Diagnostics {
	var out diag.Diagnostics
	if agent.Alias != "" {
		out.Add(warn("AF305", fmt.Sprintf("agent %q model alias %q cannot be enforced in Cursor rules; recorded in rule metadata", agent.Name, agent.Alias)))
	}
	if len(agent.Tools) > 0 {
		out.Add(warn("AF306", fmt.Sprintf("agent %q tool refs are metadata-only on Cursor; use .cursor/mcp.json for MCP servers", agent.Name)))
	}
	if agent.Permissions != "" {
		out.Add(warn("AF307", fmt.Sprintf("agent %q permissions %q cannot map to Cursor rules", agent.Name, agent.Permissions)))
	}
	return out
}

func warn(code, msg string) diag.Diagnostic {
	return diag.Diagnostic{
		Code:     code,
		Severity: diag.Warning,
		Msg:      msg,
	}
}

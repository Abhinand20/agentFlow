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
)

type needs struct {
	parallel    bool
	loopBound   bool
	outputParse bool
	blocking    []string
	hasAgents   bool
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
	}
}

// Negotiate diffs program needs vs Cursor capabilities and returns AF3xx warnings.
func Negotiate(p ir.Program) diag.Diagnostics {
	return negotiateNeeds(scanNeeds(p))
}

func scanNeeds(p ir.Program) needs {
	var n needs
	if len(p.Agents) > 0 {
		n.hasAgents = true
	}
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

func negotiateNeeds(n needs) diag.Diagnostics {
	var out diag.Diagnostics
	if n.parallel {
		out.Add(warn("AF300", "parallel spawn is sequential on cursor"))
	}
	if n.loopBound {
		out.Add(warn("AF301", "loop bound is advisory; host self-counts"))
	}
	if n.outputParse {
		out.Add(warn("AF302", "output parsing is advisory"))
	}
	for _, name := range n.blocking {
		out.Add(warn("AF303", fmt.Sprintf("gate %q falls back to advisory; cursor hooks deferred", name)))
	}
	if n.hasAgents {
		out.Add(warn("AF304", "agents emitted as .cursor/rules, not native subagents"))
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

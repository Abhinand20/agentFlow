package validate

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/flowgraph"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func errf(code string, pos lexer.Position, format string, args ...any) diag.Diagnostic {
	return diag.Diagnostic{Code: code, Severity: diag.Error, Msg: fmt.Sprintf(format, args...), Pos: pos}
}

func warnf(code string, pos lexer.Position, format string, args ...any) diag.Diagnostic {
	return diag.Diagnostic{Code: code, Severity: diag.Warning, Msg: fmt.Sprintf(format, args...), Pos: pos}
}

func walkStepRefs(steps []model.Step, visit func(ref string, pos lexer.Position)) {
	for i := range steps {
		walkStepRef(steps[i], visit)
	}
}

func walkStepRef(s model.Step, visit func(ref string, pos lexer.Position)) {
	switch s.Kind {
	case model.StepRef:
		if s.Ref != "" {
			visit(s.Ref, s.Pos)
		}
	case model.StepChain:
		walkStepRefs(s.Chain, visit)
	case model.StepBranch:
		for _, c := range s.Cases {
			walkStepRef(c.Step, visit)
		}
	case model.StepLoop, model.StepRepeat:
		walkStepRefs(s.Body, visit)
	case model.StepParallel:
		walkStepRefs(s.PFork, visit)
		if s.Gather != nil {
			walkStepRef(*s.Gather, visit)
		}
	}
}

func walkModelSteps(steps []model.Step, visit func(s model.Step)) {
	for _, st := range steps {
		walkModelStep(st, visit)
	}
}

func walkModelStep(s model.Step, visit func(s model.Step)) {
	visit(s)
	switch s.Kind {
	case model.StepChain:
		walkModelSteps(s.Chain, visit)
	case model.StepBranch:
		for _, c := range s.Cases {
			walkModelStep(c.Step, visit)
		}
	case model.StepLoop, model.StepRepeat:
		walkModelSteps(s.Body, visit)
	case model.StepParallel:
		walkModelSteps(s.PFork, visit)
		if s.Gather != nil {
			walkModelStep(*s.Gather, visit)
		}
	}
}

func walkResolved(n *flowgraph.Node, visit func(*flowgraph.Node)) {
	if n == nil {
		return
	}
	visit(n)
	switch n.Kind {
	case flowgraph.KindSeq:
		for _, s := range n.Steps {
			walkResolved(s, visit)
		}
	case flowgraph.KindLoop:
		for _, s := range n.Steps {
			walkResolved(s, visit)
		}
	case flowgraph.KindParallel:
		for _, s := range n.Steps {
			walkResolved(s, visit)
		}
		if n.Gather != nil {
			walkResolved(n.Gather, visit)
		}
	case flowgraph.KindBranch:
		for _, c := range n.Cases {
			walkResolved(c.Step, visit)
		}
	}
}

func declPos(prog *model.Program, name string) lexer.Position {
	if a := prog.Agents[name]; a != nil {
		return a.Pos
	}
	if g := prog.Gates[name]; g != nil {
		return g.Pos
	}
	if f := prog.Flows[name]; f != nil {
		return f.Pos
	}
	if t := prog.Types[name]; t != nil {
		return t.Pos
	}
	if c := prog.Capabilities[name]; c != nil {
		return c.Pos
	}
	return lexer.Position{}
}

func refNamespace(k model.DeclKind) string {
	switch k {
	case model.DeclAgent, model.DeclGate, model.DeclFlow:
		return "ref"
	case model.DeclType:
		return "type"
	case model.DeclCapability:
		return "cap"
	default:
		return "other"
	}
}

func orderedFlows(prog *model.Program) []*model.Flow {
	var out []*model.Flow
	seen := make(map[string]bool)
	for _, ref := range prog.Order {
		if ref.Kind != model.DeclFlow || seen[ref.Name] {
			continue
		}
		seen[ref.Name] = true
		if f := prog.Flows[ref.Name]; f != nil {
			out = append(out, f)
		}
	}
	return out
}

func outTypeOfRef(prog *model.Program, ref string) string {
	if a := prog.Agents[ref]; a != nil {
		if a.Out == "" {
			return "text"
		}
		return a.Out
	}
	if f := prog.Flows[ref]; f != nil {
		return f.Out
	}
	return ""
}

func valueLabelEnum(ctx ruleCtx, valueLabel string) ([]string, string) {
	if ctx.Res != nil {
		for _, inst := range ctx.Res.Instances {
			if inst.ValueLabel == valueLabel && len(inst.OutEnum) > 0 {
				return inst.OutEnum, inst.Decl
			}
		}
	}
	return nil, ""
}

func endsInBranch(steps []model.Step) bool {
	if len(steps) == 0 {
		return false
	}
	last := steps[len(steps)-1]
	if last.Kind == model.StepBranch {
		return true
	}
	if last.Kind == model.StepChain && len(last.Chain) > 0 {
		tail := last.Chain[len(last.Chain)-1]
		return tail.Kind == model.StepBranch
	}
	return false
}

func leafRef(s model.Step) (string, bool) {
	switch s.Kind {
	case model.StepRef:
		return s.Ref, s.Ref != ""
	case model.StepChain:
		if len(s.Chain) == 0 {
			return "", false
		}
		return leafRef(s.Chain[len(s.Chain)-1])
	default:
		return "", false
	}
}

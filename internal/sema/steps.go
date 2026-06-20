package sema

import (
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/ast"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func resolveSteps(steps []*ast.Step, diags *diag.Diagnostics) []model.Step {
	var out []model.Step
	for _, s := range steps {
		if s == nil {
			continue
		}
		out = append(out, resolveStep(s, diags))
	}
	return out
}

func resolveStep(s *ast.Step, diags *diag.Diagnostics) model.Step {
	switch {
	case s.Chain != nil:
		return resolveChain(s.Chain, diags)
	case s.Branch != nil:
		return resolveBranch(s.Branch, diags)
	case s.Loop != nil:
		return resolveLoop(s.Loop, diags)
	case s.Repeat != nil:
		return resolveRepeat(s.Repeat, diags)
	case s.Parallel != nil:
		return resolveParallel(s.Parallel, diags)
	default:
		return model.Step{Pos: s.Pos}
	}
}

func resolveChain(c *ast.Chain, diags *diag.Diagnostics) model.Step {
	if len(c.Atoms) == 1 {
		return resolveAtom(c.Atoms[0], diags)
	}
	st := model.Step{Kind: model.StepChain, Pos: c.Pos}
	for _, atom := range c.Atoms {
		st.Chain = append(st.Chain, resolveAtom(atom, diags))
	}
	return st
}

func resolveAtom(atom *ast.Atom, diags *diag.Diagnostics) model.Step {
	if atom == nil {
		return model.Step{}
	}
	if atom.Args != nil || atom.Block != nil {
		return resolveCall(atom, diags)
	}
	st := model.Step{Kind: model.StepRef, Pos: atom.Pos}
	if atom.Name != nil {
		st.Ref = qualNameStr(atom.Name)
	}
	if atom.Alias != nil {
		st.Alias = *atom.Alias
	}
	return st
}

func resolveCall(atom *ast.Atom, diags *diag.Diagnostics) model.Step {
	st := model.Step{
		Kind: model.StepCall,
		Pos:  atom.Pos,
	}
	if atom.Name != nil {
		st.Ref = qualNameStr(atom.Name)
	}
	if atom.Args != nil {
		st.CallArgs = atom.Args.Items
	}
	if atom.Block != nil {
		st.CallBlock = resolveSteps(atom.Block, diags)
	}
	if atom.Alias != nil {
		st.Alias = *atom.Alias
	}
	return st
}

func resolveBranch(b *ast.Branch, diags *diag.Diagnostics) model.Step {
	st := model.Step{Kind: model.StepBranch, Pos: b.Pos}
	if b.Value != nil {
		st.BranchIsIt = b.Value.It
		if b.Value.Name != nil {
			st.BranchValue = qualNameStr(b.Value.Name)
		}
	}
	for _, c := range b.Cases {
		if c == nil {
			continue
		}
		cs := model.Case{Values: c.Values, Pos: c.Pos}
		if c.Step != nil {
			cs.Step = resolveStep(c.Step, diags)
		}
		st.Cases = append(st.Cases, cs)
	}
	return st
}

func resolveLoop(l *ast.Loop, diags *diag.Diagnostics) model.Step {
	return resolveLoopBody(l.Body, l.Cond, l.Max, false, l.Pos, diags)
}

func resolveRepeat(r *ast.Repeat, diags *diag.Diagnostics) model.Step {
	return resolveLoopBody(r.Body, r.Cond, r.Max, true, r.Pos, diags)
}

func resolveLoopBody(body []*ast.Step, cond *ast.Cond, max *float64, doWhile bool, pos lexer.Position, diags *diag.Diagnostics) model.Step {
	kind := model.StepLoop
	if doWhile {
		kind = model.StepRepeat
	}
	st := model.Step{
		Kind:    kind,
		Pos:     pos,
		DoWhile: doWhile,
		Body:    resolveSteps(body, diags),
	}
	if cond != nil {
		st.Cond = resolveCond(cond)
	}
	if max != nil {
		st.HasMax = true
		if n, ok := positiveInt(*max); ok {
			st.Max = &n
		} else {
			diags.Add(diag.Diagnostic{
				Code:     "AF000",
				Severity: diag.Error,
				Msg:      "max must be a positive integer",
				Pos:      pos,
			})
		}
	}
	return st
}

func resolveCond(c *ast.Cond) *model.Cond {
	mc := &model.Cond{Op: c.Op, Enum: c.Enum, Pos: c.Pos}
	if c.Value != nil {
		mc.IsIt = c.Value.It
		if c.Value.Name != nil {
			mc.Value = qualNameStr(c.Value.Name)
		}
	}
	return mc
}

func resolveParallel(p *ast.Parallel, diags *diag.Diagnostics) model.Step {
	st := model.Step{Kind: model.StepParallel, Pos: p.Pos}
	st.PFork = resolveSteps(p.Body, diags)
	if p.Gather != nil {
		gather := resolveAtom(p.Gather, diags)
		st.Gather = &gather
	}
	if p.Each != nil {
		st.Each = &model.Each{
			Item: qualNameStr(p.Each.Item),
			As:   p.Each.As,
			Pos:  p.Each.Pos,
		}
	}
	return st
}

func positiveInt(n float64) (int, bool) {
	if n != float64(int(n)) || n <= 0 {
		return 0, false
	}
	return int(n), true
}

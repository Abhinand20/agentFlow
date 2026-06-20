package validate

import (
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleReachability(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	prog := ctx.Prog
	if prog.EntryFlow == "" {
		return out
	}
	reached := make(map[string]bool)
	var walkFlow func(name string)
	walkFlow = func(name string) {
		if reached[name] {
			return
		}
		reached[name] = true
		f := prog.Flows[name]
		if f == nil {
			return
		}
		walkStepRefs(f.Body, func(ref string, _ lexer.Position) {
			if prog.Flows[ref] != nil {
				walkFlow(ref)
			}
		})
	}
	walkFlow(prog.EntryFlow)

	for _, ref := range prog.Order {
		if ref.Kind != model.DeclFlow {
			continue
		}
		if !reached[ref.Name] {
			f := prog.Flows[ref.Name]
			if f != nil {
				out.Add(warnf("AF207", f.Pos, "flow %s is never reached from entry", f.Name))
			}
		}
	}
	return out
}

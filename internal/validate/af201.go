package validate

import (
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleNodeResolution(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	prog := ctx.Prog
	terminals := model.Terminals()

	for _, f := range orderedFlows(prog) {
		walkStepRefs(f.Body, func(ref string, pos lexer.Position) {
			if terminals[ref] {
				return
			}
			if prog.Agents[ref] != nil || prog.Gates[ref] != nil || prog.Flows[ref] != nil {
				return
			}
			out.Add(errf("AF201", pos, "unknown step %q in flow %s", ref, f.Name))
		})
	}
	return out
}

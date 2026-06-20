package validate

import (
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleTypesExist(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	prog := ctx.Prog

	for _, ref := range prog.Order {
		switch ref.Kind {
		case model.DeclAgent:
			a := prog.Agents[ref.Name]
			if a == nil {
				continue
			}
			if a.Out != "" && a.Out != "text" && prog.Types[a.Out] == nil {
				out.Add(errf("AF202", a.Pos, "agent %s: out type %q is not a declared enum", a.Name, a.Out))
			}
			if a.In != "" && a.In != "text" && prog.Types[a.In] == nil {
				// in: opaque nominal is allowed — only explicit bad enum refs on out fail
				_ = a.In
			}
		case model.DeclFlow:
			f := prog.Flows[ref.Name]
			if f == nil {
				continue
			}
			if f.OutExplicit && f.Out != "" && f.Out != "text" && prog.Types[f.Out] == nil {
				out.Add(errf("AF202", f.Pos, "flow %s: out type %q is not a declared enum", f.Name, f.Out))
			}
		}
	}
	return out
}

package validate

import (
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleToolRefs(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	prog := ctx.Prog
	for _, ref := range prog.Order {
		if ref.Kind != model.DeclAgent {
			continue
		}
		a := prog.Agents[ref.Name]
		if a == nil {
			continue
		}
		for _, tr := range a.Tools {
			if tr.Tool == "" {
				continue // unqualified — not checked in v0.1
			}
			cap := prog.Capabilities[tr.Capability]
			if cap == nil {
				out.Add(errf("AF206", tr.Pos,
					"tool %q: capability %q not found", tr.Raw, tr.Capability))
				continue
			}
			found := false
			for _, t := range cap.Tools {
				if t == tr.Tool {
					found = true
					break
				}
			}
			if !found {
				out.Add(errf("AF206", tr.Pos,
					"tool %q not provided by capability %s", tr.Raw, tr.Capability))
			}
		}
	}
	return out
}

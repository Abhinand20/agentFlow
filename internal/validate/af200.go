package validate

import (
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleDuplicateNames(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	prog := ctx.Prog
	terminals := model.Terminals()

	seen := make(map[string]bool)
	reported := make(map[string]bool)
	reservedReported := make(map[string]bool)

	for _, ref := range prog.Order {
		ns := refNamespace(ref.Kind)
		key := ns + "\x00" + ref.Name

		if ns == "ref" && terminals[ref.Name] && !reservedReported[ref.Name] {
			reservedReported[ref.Name] = true
			out.Add(errf("AF200", declPos(prog, ref.Name),
				"duplicate declaration %q (redefines reserved terminal)", ref.Name))
		}

		if seen[key] {
			if !reported[key] {
				reported[key] = true
				out.Add(errf("AF200", declPos(prog, ref.Name),
					"duplicate declaration %q", ref.Name))
			}
			continue
		}
		seen[key] = true
	}
	return out
}

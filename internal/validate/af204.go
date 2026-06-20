package validate

import (
	"strings"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/flowgraph"
)

func ruleBranchExhaustive(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	if ctx.Res == nil {
		return out
	}
	walkResolved(ctx.Res.Tree, func(n *flowgraph.Node) {
		if n.Kind != flowgraph.KindBranch {
			return
		}
		members, _ := valueLabelEnum(ctx, n.BranchValue)
		if len(members) == 0 {
			return
		}
		covered := make(map[string]bool)
		for _, c := range n.Cases {
			for _, v := range c.Values {
				covered[v] = true
			}
		}
		var missing []string
		for _, m := range members {
			if !covered[m] {
				missing = append(missing, m)
			}
		}
		if len(missing) > 0 {
			out.Add(warnf("AF204", posForValue(ctx, n.BranchValue),
				"branch on %s may not handle: %s", n.BranchValue, strings.Join(missing, ", ")))
		}
	})
	return out
}

package validate

import (
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/flowgraph"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleConditionEnum(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	if ctx.Res == nil {
		return out
	}
	walkResolved(ctx.Res.Tree, func(n *flowgraph.Node) {
		switch n.Kind {
		case flowgraph.KindBranch:
			checkBranchEnum(ctx, n, &out)
		case flowgraph.KindLoop:
			if n.Cond != nil {
				checkCondEnum(ctx, n.Cond, n, &out)
			}
		}
	})
	return out
}

func checkBranchEnum(ctx ruleCtx, n *flowgraph.Node, out *diag.Diagnostics) {
	members, decl := valueLabelEnum(ctx, n.BranchValue)
	if len(members) == 0 {
		out.Add(errf("AF203", posForValue(ctx, n.BranchValue),
			"branch reads %q which has no enum output", n.BranchValue))
		return
	}
	memberSet := make(map[string]bool, len(members))
	for _, m := range members {
		memberSet[m] = true
	}
	for _, c := range n.Cases {
		for _, v := range c.Values {
			if !memberSet[v] {
				out.Add(errf("AF203", posForValue(ctx, n.BranchValue),
					"branch reads %q (%s); %q is not a member", n.BranchValue, decl, v))
			}
		}
	}
}

func checkCondEnum(ctx ruleCtx, c *model.Cond, n *flowgraph.Node, out *diag.Diagnostics) {
	if c == nil || c.Enum == "" {
		return
	}
	members, decl := valueLabelEnum(ctx, c.Value)
	if len(members) == 0 {
		out.Add(errf("AF203", c.Pos,
			"condition reads %q which has no enum output", c.Value))
		return
	}
	found := false
	for _, m := range members {
		if m == c.Enum {
			found = true
			break
		}
	}
	if !found {
		out.Add(errf("AF203", c.Pos,
			"condition reads %q (%s); %q is not a member", c.Value, decl, c.Enum))
	}
	_ = n
}

func posForValue(ctx ruleCtx, valueLabel string) lexer.Position {
	if ctx.Res != nil {
		for _, inst := range ctx.Res.Instances {
			if inst.ValueLabel == valueLabel {
				if a := ctx.Prog.Agents[inst.Decl]; a != nil {
					return a.Pos
				}
			}
		}
	}
	return lexer.Position{}
}

package validate

import (
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func rulePromptSource(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	for _, ref := range ctx.Prog.Order {
		if ref.Kind != model.DeclAgent {
			continue
		}
		a := ctx.Prog.Agents[ref.Name]
		if a == nil || a.Resolution.Reason == "" {
			continue
		}
		switch a.Resolution.Reason {
		case "conflict":
			out.Add(errf("AF211", a.Pos,
				"agent %s: prompt and prompt-file are mutually exclusive", a.Name))
		case "missing":
			out.Add(errf("AF211", a.Pos,
				"agent %s: prompt file %q not found", a.Name, a.Resolution.Path))
		case "escapes-tree":
			out.Add(errf("AF211", a.Pos,
				"agent %s: prompt path %q escapes source directory", a.Name, a.Resolution.Path))
		case "absolute":
			out.Add(errf("AF211", a.Pos,
				"agent %s: prompt path %q must be relative", a.Name, a.Resolution.Path))
		case "unreadable":
			out.Add(errf("AF211", a.Pos,
				"agent %s: prompt file %q is unreadable", a.Name, a.Resolution.Path))
		case "not-utf8":
			out.Add(errf("AF211", a.Pos,
				"agent %s: prompt file %q is not valid UTF-8", a.Name, a.Resolution.Path))
		default:
			out.Add(errf("AF211", a.Pos,
				"agent %s: invalid prompt source (%s)", a.Name, a.Resolution.Reason))
		}
	}
	return out
}

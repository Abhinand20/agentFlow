package validate

import (
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleCycleBounded(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	for _, f := range orderedFlows(ctx.Prog) {
		walkModelSteps(f.Body, func(s model.Step) {
			if s.Kind != model.StepLoop && s.Kind != model.StepRepeat {
				return
			}
			if !s.HasMax {
				out.Add(errf("AF205", s.Pos, "loop has no max bound"))
			}
		})
	}
	return out
}

package validate

import (
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleBranchTerminalOut(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	for _, f := range orderedFlows(ctx.Prog) {
		if f.ReturnExplicit || !endsInBranch(f.Body) {
			continue
		}
		if !f.OutExplicit || f.Out == "" {
			continue
		}
		checkBranchLeaves(f.Body, f.Out, f.Name, f.Pos, ctx, &out)
	}
	return out
}

func checkBranchLeaves(steps []model.Step, wantOut, flowName string, flowPos lexer.Position, ctx ruleCtx, out *diag.Diagnostics) {
	walkModelSteps(steps, func(s model.Step) {
		if s.Kind != model.StepBranch {
			return
		}
		for _, c := range s.Cases {
			ref, ok := leafRef(c.Step)
			if !ok {
				continue
			}
			got := outTypeOfRef(ctx.Prog, ref)
			if got == "" {
				got = "text"
			}
			want := wantOut
			if want == "" {
				want = "text"
			}
			if got != want {
				out.Add(errf("AF210", flowPos,
					"flow %s: branch leaf %s out %q != flow out %q", flowName, ref, got, want))
			}
		}
	})
}

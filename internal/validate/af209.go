package validate

import (
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleReturnBinding(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	for _, f := range orderedFlows(ctx.Prog) {
		if !f.ReturnExplicit {
			continue
		}
		if endsInBranch(f.Body) {
			continue
		}
		producer, ok := returnProducer(f.Body, f.Return)
		if !ok {
			out.Add(errf("AF209", f.Pos,
				"flow %s: return %q is not a value label", f.Name, f.Return))
			continue
		}
		if f.OutExplicit && f.Out != "" {
			got := outTypeOfRef(ctx.Prog, producer)
			if got == "" {
				got = "text"
			}
			if got != f.Out {
				out.Add(errf("AF209", f.Pos,
					"flow %s: return %q type %q does not match flow out %q",
					f.Name, f.Return, got, f.Out))
			}
		}
	}
	return out
}

// returnProducer finds the declaration ref that binds the given value label in a flow body.
func returnProducer(steps []model.Step, label string) (string, bool) {
	var found string
	ok := false
	walkModelSteps(steps, func(s model.Step) {
		if ok {
			return
		}
		switch s.Kind {
		case model.StepRef:
			if s.Alias == label {
				found = s.Ref
				ok = true
			}
		case model.StepParallel:
			if s.Gather != nil && s.Gather.Alias == label {
				found = s.Gather.Ref
				ok = true
			}
		}
	})
	return found, ok
}

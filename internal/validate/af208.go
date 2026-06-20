package validate

import (
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func ruleDuplicateLabels(ctx ruleCtx) diag.Diagnostics {
	var out diag.Diagnostics
	for _, f := range orderedFlows(ctx.Prog) {
		checkSeqScope(f.Body, f.Name, &out)
	}
	return out
}

// checkSeqScope flags duplicate bare refs within one sequential step list.
// Nested scopes (loop bodies, branch arms, parallel forks) are checked separately.
func checkSeqScope(steps []model.Step, flowName string, out *diag.Diagnostics) {
	counts := make(map[string]int)
	firstPos := make(map[string]lexer.Position)
	for _, st := range steps {
		recordBareRef(st, counts, firstPos)
		checkNestedScopes(st, flowName, out)
	}
	for ref, n := range counts {
		if n > 1 {
			out.Add(errf("AF208", firstPos[ref],
				"two steps named %q in flow %s; use \"as\" to disambiguate", ref, flowName))
		}
	}
}

func checkNestedScopes(st model.Step, flowName string, out *diag.Diagnostics) {
	switch st.Kind {
	case model.StepLoop, model.StepRepeat:
		checkSeqScope(st.Body, flowName, out)
	case model.StepBranch:
		for _, c := range st.Cases {
			checkSeqScope([]model.Step{c.Step}, flowName, out)
		}
	case model.StepParallel:
		for _, fork := range st.PFork {
			checkSeqScope([]model.Step{fork}, flowName, out)
		}
		if st.Gather != nil {
			checkSeqScope([]model.Step{*st.Gather}, flowName, out)
		}
	case model.StepChain:
		checkSeqScope(st.Chain, flowName, out)
	}
}

func recordBareRef(st model.Step, counts map[string]int, firstPos map[string]lexer.Position) {
	switch st.Kind {
	case model.StepRef:
		if st.Ref != "" && st.Alias == "" {
			if counts[st.Ref] == 0 {
				firstPos[st.Ref] = st.Pos
			}
			counts[st.Ref]++
		}
	case model.StepChain:
		for _, c := range st.Chain {
			recordBareRef(c, counts, firstPos)
		}
	}
}

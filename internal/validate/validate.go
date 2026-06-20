package validate

import (
	"sort"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/flowgraph"
	"github.com/Abhinand20/agentFlow/internal/model"
)

type ruleCtx struct {
	Prog *model.Program
	Res  *flowgraph.Resolved
}

type rule struct {
	code string
	fn   func(ruleCtx) diag.Diagnostics
}

// AF208 (parallel duplicate label, warning), AF209 (missing default return), and
// AF212 (recursive nesting) are emitted by M3 during flowgraph.Resolve. M4 does
// not re-emit them; the pipeline concatenates M3 + M4 diagnostics. M4's AF209
// covers only explicit return checks M3 does not perform.
var rules = []rule{
	{"AF200", ruleDuplicateNames},
	{"AF201", ruleNodeResolution},
	{"AF202", ruleTypesExist},
	{"AF203", ruleConditionEnum},
	{"AF204", ruleBranchExhaustive},
	{"AF205", ruleCycleBounded},
	{"AF206", ruleToolRefs},
	{"AF207", ruleReachability},
	{"AF208", ruleDuplicateLabels},
	{"AF209", ruleReturnBinding},
	{"AF210", ruleBranchTerminalOut},
	{"AF211", rulePromptSource},
}

// Validate runs the v0.1 rule set over the resolved flow plus the model.
func Validate(prog *model.Program, res *flowgraph.Resolved) diag.Diagnostics {
	var out diag.Diagnostics
	if prog == nil {
		return out
	}
	ctx := ruleCtx{Prog: prog, Res: res}
	for _, r := range rules {
		out.Add(r.fn(ctx)...)
	}
	sortDiagnostics(out)
	return out
}

func sortDiagnostics(d diag.Diagnostics) {
	sort.SliceStable(d, func(i, j int) bool {
		a, b := d[i], d[j]
		if a.Pos.Filename != b.Pos.Filename {
			return a.Pos.Filename < b.Pos.Filename
		}
		if a.Pos.Line != b.Pos.Line {
			return a.Pos.Line < b.Pos.Line
		}
		if a.Pos.Column != b.Pos.Column {
			return a.Pos.Column < b.Pos.Column
		}
		return a.Code < b.Code
	})
}

package render

import (
	"fmt"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/ir"
)

type runbookCtx struct {
	p    ir.Program
	v    Vocabulary
	out  *strings.Builder
	step int
}

// Runbook renders the entry flow as a numbered runbook markdown body.
func Runbook(p ir.Program, v Vocabulary) string {
	ctx := &runbookCtx{p: p, v: v, out: &strings.Builder{}}
	ctx.renderNode(p.Flow.Root, 0)
	return strings.TrimRight(ctx.out.String(), "\n")
}

func (ctx *runbookCtx) renderNode(n ir.Node, indent int) {
	switch n.Kind {
	case ir.NodeSeq:
		for _, child := range n.Steps {
			ctx.renderNode(child, indent)
		}
	case ir.NodeStep:
		ctx.renderStep(n, indent)
	case ir.NodeBranch:
		ctx.renderBranch(n, indent)
	case ir.NodeLoop:
		if n.DoWhile {
			ctx.renderRepeat(n, indent)
		} else {
			ctx.renderLoop(n, indent)
		}
	case ir.NodeParallel:
		ctx.renderParallel(n, indent)
	}
}

func (ctx *runbookCtx) indentPrefix(indent int) string {
	return strings.Repeat("  ", indent)
}

func (ctx *runbookCtx) renderStep(n ir.Node, indent int) {
	if n.Step == nil {
		return
	}
	step := n.Step
	prefix := ctx.indentPrefix(indent)

	if indent == 0 {
		ctx.step++
		line := fmt.Sprintf("%d. ", ctx.step)
		ctx.out.WriteString(prefix + line)
	} else {
		ctx.out.WriteString(prefix + "- ")
	}

	switch step.Kind {
	case "agent":
		agent := ctx.findAgent(step.Decl)
		view := AgentView{
			Name:         agent.Name,
			Decl:         step.Decl,
			ControlLabel: step.ControlLabel,
			In:           agent.In,
			OutEnum:      step.OutEnum,
			PrevProducer: step.PrevProducer,
			UsesFlowArg:  ctx.usesFlowArg(agent),
		}
		ctx.out.WriteString(ctx.v.InvokeAgent(view))
		if len(step.OutEnum) > 0 {
			retry := agent.Retry
			ctx.out.WriteString(" ")
			ctx.out.WriteString(ctx.v.ParseOutputProtocol(step.OutEnum, retry))
		}
	case "gate":
		gate := ctx.findGate(step.Decl)
		gv := GateView{
			Name:         gate.Name,
			Run:          gate.Run,
			OnFail:       gate.OnFail,
			OnFailTarget: gate.OnFailTarget,
			ScriptRetry:  gate.ScriptRetry,
		}
		ctx.out.WriteString(ctx.v.RunScript(gv))
		ctx.out.WriteString(" ")
		ctx.out.WriteString(ctx.gateOnFailProse(gv))
	}
	ctx.out.WriteString("\n")
}

func (ctx *runbookCtx) usesFlowArg(a ir.Agent) bool {
	if ctx.p.Entry.InType == "" || a.In == "" {
		return false
	}
	return a.In == ctx.p.Entry.InType
}

func (ctx *runbookCtx) gateOnFailProse(g GateView) string {
	switch g.OnFail {
	case "halt":
		return "If the gate fails, stop the flow and report failure."
	case "retry":
		gotoStep := strings.TrimSuffix(ctx.v.GotoStep(g.OnFailTarget), ".")
		return fmt.Sprintf(
			"If the gate fails, %s and re-run from there. Retry the script up to `%d` times first.",
			gotoStep,
			g.ScriptRetry,
		)
	case "goto":
		return fmt.Sprintf("If the gate fails, jump to step `%s`.", g.OnFailTarget)
	case "enter-loop":
		return "If the gate fails, re-enter the current loop."
	default:
		return "If the gate fails, stop the flow and report failure."
	}
}

func (ctx *runbookCtx) renderBranch(n ir.Node, indent int) {
	prefix := ctx.indentPrefix(indent)
	if indent == 0 {
		ctx.step++
		ctx.out.WriteString(fmt.Sprintf("%s%d. ", prefix, ctx.step))
	} else {
		ctx.out.WriteString(prefix)
	}
	ctx.out.WriteString(ctx.v.ReadOutput(n.BranchValue))
	ctx.out.WriteString(" Then:\n")

	for _, c := range n.Cases {
		casePrefix := ctx.indentPrefix(indent + 1)
		values := strings.Join(c.Values, ", ")
		ctx.out.WriteString(fmt.Sprintf("%s- if `%s` is `%s`, do:\n", casePrefix, n.BranchValue, values))
		ctx.renderNode(c.Body, indent+2)
	}
}

func (ctx *runbookCtx) renderLoop(n ir.Node, indent int) {
	prefix := ctx.indentPrefix(indent)
	if indent == 0 {
		ctx.step++
		ctx.out.WriteString(fmt.Sprintf("%s%d. ", prefix, ctx.step))
	} else {
		ctx.out.WriteString(prefix)
	}
	cond := ctx.formatWhileCond(n.Cond)
	maxPart := ""
	if n.HasMax {
		maxPart = fmt.Sprintf(", at most `%d` times (advisory — track your iteration count and stop at %d)", n.Max, n.Max)
	}
	ctx.out.WriteString(fmt.Sprintf("Repeat the following steps while %s%s:\n", cond, maxPart))
	if n.Body != nil {
		ctx.renderNode(*n.Body, indent+1)
	}
}

func (ctx *runbookCtx) renderRepeat(n ir.Node, indent int) {
	prefix := ctx.indentPrefix(indent)
	if indent == 0 {
		ctx.step++
		ctx.out.WriteString(fmt.Sprintf("%s%d.\n", prefix, ctx.step))
	} else {
		ctx.out.WriteString(prefix)
	}
	if n.Body != nil {
		ctx.renderNode(*n.Body, indent+1)
	}
	cond := ctx.formatUntilCond(n.Cond)
	maxPart := ""
	if n.HasMax {
		maxPart = fmt.Sprintf(", at most `%d` times", n.Max)
	}
	repeatPrefix := ctx.indentPrefix(indent)
	ctx.out.WriteString(fmt.Sprintf(
		"%sThen repeat the steps above until %s%s. The check happens **after** each pass, so always run the steps at least once.",
		repeatPrefix,
		cond,
		maxPart,
	))
	ctx.out.WriteString("\n")
}

func (ctx *runbookCtx) renderParallel(n ir.Node, indent int) {
	prefix := ctx.indentPrefix(indent)
	if indent == 0 {
		ctx.step++
		ctx.out.WriteString(fmt.Sprintf("%s%d. ", prefix, ctx.step))
	} else {
		ctx.out.WriteString(prefix)
	}

	views := make([]StepView, 0, len(n.Branches))
	for _, b := range n.Branches {
		if b.Step != nil {
			views = append(views, StepView{
				ControlLabel: b.Step.ControlLabel,
				Decl:         b.Step.Decl,
				Kind:         b.Step.Kind,
			})
		}
	}
	ctx.out.WriteString(ctx.v.SpawnParallel(views))
	ctx.out.WriteString("\n")

	if n.Gather != nil {
		gatherPrefix := prefix
		if indent == 0 {
			ctx.step++
			ctx.out.WriteString(fmt.Sprintf("%s%d. ", gatherPrefix, ctx.step))
		} else {
			ctx.out.WriteString(gatherPrefix + "- ")
		}
		branchNames := make([]string, 0)
		if n.Payload != nil {
			for _, b := range n.Payload.Branches {
				branchNames = append(branchNames, b.ValueLabel)
			}
		}
		ctx.out.WriteString(fmt.Sprintf(
			"Collect the outputs from `%s` and give them to `%s`.",
			strings.Join(branchNames, "`, `"),
			n.Gather.Decl,
		))
		if n.Gather.PrevProducer != "" {
			ctx.out.WriteString(fmt.Sprintf(" Use the output from `%s` as sequential context.", n.Gather.PrevProducer))
		}
		ctx.out.WriteString("\n")
	}
}

func (ctx *runbookCtx) formatUntilCond(c *ir.Cond) string {
	if c == nil {
		return ""
	}
	switch c.Op {
	case "!=":
		return fmt.Sprintf("`%s` is not `%s`", c.ValueLabel, c.Enum)
	case "==":
		return fmt.Sprintf("`%s` equals `%s`", c.ValueLabel, c.Enum)
	default:
		return fmt.Sprintf("`%s` %s `%s`", c.ValueLabel, c.Op, c.Enum)
	}
}

func (ctx *runbookCtx) formatWhileCond(c *ir.Cond) string {
	if c == nil {
		return ""
	}
	switch c.Op {
	case "!=":
		return fmt.Sprintf("`%s` equals `%s`", c.ValueLabel, c.Enum)
	case "==":
		return fmt.Sprintf("`%s` is not `%s`", c.ValueLabel, c.Enum)
	default:
		return fmt.Sprintf("`%s` %s `%s`", c.ValueLabel, c.Op, c.Enum)
	}
}

func (ctx *runbookCtx) findAgent(decl string) ir.Agent {
	for _, a := range ctx.p.Agents {
		if a.Name == decl {
			return a
		}
	}
	return ir.Agent{Name: decl}
}

func (ctx *runbookCtx) findGate(decl string) ir.Gate {
	for _, g := range ctx.p.Gates {
		if g.Name == decl {
			return g
		}
	}
	return ir.Gate{Name: decl}
}

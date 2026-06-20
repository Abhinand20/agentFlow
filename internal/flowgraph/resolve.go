package flowgraph

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

// Resolve lowers a semantic model into a normalized flow graph for the entry flow.
func Resolve(prog *model.Program) (*Resolved, diag.Diagnostics) {
	if prog == nil || prog.EntryFlow == "" {
		return nil, nil
	}
	b := &builder{
		prog:      prog,
		instances: make(map[string]*StepInstance),
	}
	entry := prog.Flows[prog.EntryFlow]
	if entry == nil {
		return nil, b.diags
	}
	tree := b.lowerSteps(entry.Body, scope{})
	if tree == nil {
		tree = &Node{Kind: KindSeq}
	} else if tree.Kind != KindSeq {
		tree = &Node{Kind: KindSeq, Steps: []*Node{tree}}
	}
	b.wireSequentialEdges(tree)
	b.resolveFlowReturn(entry, tree, scope{})
	return &Resolved{
		Entry:     prog.EntryFlow,
		Tree:      tree,
		Instances: b.instances,
	}, b.diags
}

type scope struct {
	prefix     string
	loopPrefix string
	loopCount  int
}

type builder struct {
	prog      *model.Program
	diags     diag.Diagnostics
	instances map[string]*StepInstance
	visiting  []string
}

func (b *builder) lowerSteps(steps []model.Step, sc scope) *Node {
	if len(steps) == 0 {
		return nil
	}
	var nodes []*Node
	for _, st := range steps {
		n := b.lowerStep(st, sc)
		if n == nil {
			continue
		}
		if n.Kind == KindSeq {
			nodes = append(nodes, n.Steps...)
		} else {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		return nil
	}
	if len(nodes) == 1 {
		return nodes[0]
	}
	return &Node{Kind: KindSeq, Steps: nodes}
}

func (b *builder) lowerStep(st model.Step, sc scope) *Node {
	switch st.Kind {
	case model.StepRef:
		return b.lowerRef(st, sc)
	case model.StepChain:
		return b.lowerSteps(st.Chain, sc)
	case model.StepBranch:
		return b.lowerBranch(st, sc)
	case model.StepLoop, model.StepRepeat:
		return b.lowerLoop(st, sc)
	case model.StepParallel:
		return b.lowerParallel(st, sc)
	default:
		return nil
	}
}

func (b *builder) lowerRef(st model.Step, sc scope) *Node {
	if st.Ref == "" {
		return nil
	}
	if f, ok := b.prog.Flows[st.Ref]; ok {
		return b.inlineFlow(st, f, sc)
	}
	return b.registerLeaf(st, sc)
}

func (b *builder) inlineFlow(st model.Step, f *model.Flow, sc scope) *Node {
	subLabel := st.Ref
	if st.Alias != "" {
		subLabel = st.Alias
	}
	for _, v := range b.visiting {
		if v == f.Name {
			b.diags.Add(diag.Diagnostic{
				Code:     "AF212",
				Severity: diag.Error,
				Msg:      fmt.Sprintf("recursive flow nesting: %s", strings.Join(append(b.visiting, f.Name), " -> ")),
				Pos:      st.Pos,
			})
			return nil
		}
	}
	b.visiting = append(b.visiting, f.Name)
	childSc := scope{prefix: sc.prefix + subLabel + "."}
	body := b.lowerSteps(f.Body, childSc)
	b.visiting = b.visiting[:len(b.visiting)-1]

	b.registerSubflowValue(subLabel, f, sc, st.Pos)

	if body == nil {
		return nil
	}
	return body
}

func (b *builder) registerSubflowValue(subLabel string, f *model.Flow, sc scope, pos lexer.Position) {
	control := sc.prefix + subLabel
	valueLabel := control
	returnsFrom := b.resolveReturnProducer(f, scope{prefix: sc.prefix + subLabel + "."}, pos)
	var outEnum []string
	if f.Out != "" && f.Out != "text" {
		if et, ok := b.prog.Types[f.Out]; ok {
			outEnum = append([]string(nil), et.Values...)
		}
	}
	inst := &StepInstance{
		ControlLabel: control,
		ValueLabel:   valueLabel,
		Decl:         f.Name,
		Kind:         model.StepRef,
		OutEnum:      outEnum,
		ReturnsFrom:  returnsFrom,
	}
	b.instances[control] = inst
}

func (b *builder) registerLeaf(st model.Step, sc scope) *Node {
	decl := st.Ref
	control := b.allocControlLabel(sc.prefix+sc.loopPrefix+decl, sc)
	value := control
	if st.Alias != "" {
		value = sc.prefix + st.Alias
	}

	var outEnum []string
	if a, ok := b.prog.Agents[decl]; ok {
		if a.Out != "" {
			if et, ok := b.prog.Types[a.Out]; ok {
				outEnum = append([]string(nil), et.Values...)
			}
		}
	} else if _, ok := b.prog.Gates[decl]; ok {
		value = "" // gates produce no output
	}

	inst := &StepInstance{
		ControlLabel: control,
		ValueLabel:   value,
		Decl:         decl,
		Kind:         model.StepRef,
		OutEnum:      outEnum,
	}
	b.instances[control] = inst
	return &Node{Kind: KindStep, Label: control}
}

func (b *builder) lowerBranch(st model.Step, sc scope) *Node {
	branchSc := sc
	branchSc.loopPrefix = ""
	branchSc.loopCount = 0
	n := &Node{
		Kind:        KindBranch,
		BranchValue: sc.prefix + st.BranchValue,
	}
	seen := make(map[string]string) // dedupe key -> control label (D1)
	for _, c := range st.Cases {
		step := b.lowerCaseStep(c.Step, branchSc, seen)
		n.Cases = append(n.Cases, &ResolvedCase{Values: append([]string(nil), c.Values...), Step: step})
	}
	return n
}

func (b *builder) lowerCaseStep(st model.Step, sc scope, seen map[string]string) *Node {
	if st.Kind != model.StepRef {
		return b.lowerStep(st, sc)
	}
	key := st.Ref + "\x00" + st.Alias
	if label, ok := seen[key]; ok {
		return &Node{Kind: KindStep, Label: label}
	}
	n := b.lowerRef(st, sc)
	if n != nil && n.Kind == KindStep {
		seen[key] = n.Label
	}
	return n
}

func (b *builder) lowerLoop(st model.Step, sc scope) *Node {
	loopSc := sc
	loopSc.loopCount++
	if st.DoWhile {
		loopSc.loopPrefix = "repeat."
		if loopSc.loopCount > 1 {
			loopSc.loopPrefix = fmt.Sprintf("repeat%d.", loopSc.loopCount)
		}
	} else {
		loopSc.loopPrefix = "loop."
		if loopSc.loopCount > 1 {
			loopSc.loopPrefix = fmt.Sprintf("loop%d.", loopSc.loopCount)
		}
	}
	body := b.lowerSteps(st.Body, loopSc)
	var max *int
	if st.HasMax && st.Max != nil {
		v := *st.Max
		max = &v
	}
	var cond *model.Cond
	if st.Cond != nil {
		c := *st.Cond
		if c.Value != "" {
			c.Value = loopSc.prefix + c.Value
		}
		cond = &c
	}
	return &Node{
		Kind:    KindLoop,
		Steps:   bodySteps(body),
		Cond:    cond,
		Max:     max,
		DoWhile: st.DoWhile,
	}
}

func bodySteps(body *Node) []*Node {
	if body == nil {
		return nil
	}
	if body.Kind == KindSeq {
		return body.Steps
	}
	return []*Node{body}
}

func (b *builder) lowerParallel(st model.Step, sc scope) *Node {
	parSc := sc
	parSc.prefix = "" // parallel branches use unprefixed control labels (§4.8)
	parSc.loopPrefix = ""
	var branches []*Node
	payload := make(map[string]string)
	for _, fork := range st.PFork {
		n := b.lowerStep(fork, parSc)
		if n == nil {
			continue
		}
		branches = append(branches, n)
		if n.Kind == KindStep {
			if inst := b.instances[n.Label]; inst != nil {
				key := inst.Decl
				val := inst.ValueLabel
				if val == "" {
					val = inst.ControlLabel
				}
				payload[key] = val
			}
		}
	}
	// gather step keeps enclosing scope prefix
	var gather *Node
	if st.Gather != nil {
		gather = b.lowerRef(*st.Gather, sc)
		if gather != nil && gather.Kind == KindStep {
			if inst := b.instances[gather.Label]; inst != nil {
				inst.GatherPayload = copyMap(payload)
			}
		}
	}
	return &Node{
		Kind:          KindParallel,
		Steps:         branches,
		Gather:        gather,
		GatherPayload: copyMap(payload),
	}
}

func (b *builder) allocControlLabel(label string, _ scope) string {
	if _, exists := b.instances[label]; !exists {
		return label
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s#%d", label, i)
		if _, exists := b.instances[candidate]; !exists {
			return candidate
		}
	}
}

func (b *builder) resolveFlowReturn(f *model.Flow, tree *Node, sc scope) {
	if len(f.Body) == 0 {
		return
	}
	if f.ReturnExplicit {
		return // explicit return validated in M4
	}
	// Rule 0: default return = terminal producer
	terminal := lastSequentialProducer(tree)
	if terminal == "" {
		if endsInBranch(tree) {
			return // branch-terminal; M4 AF210
		}
		b.diags.Add(diag.Diagnostic{
			Code:     "AF209",
			Severity: diag.Error,
			Msg:      fmt.Sprintf("flow %q: cannot infer default return (no terminal producer)", f.Name),
			Pos:      f.Pos,
		})
		return
	}
	inst := b.instances[terminal]
	if inst == nil || inst.ValueLabel == "" {
		b.diags.Add(diag.Diagnostic{
			Code:     "AF209",
			Severity: diag.Error,
			Msg:      fmt.Sprintf("flow %q: terminal step %q produces no output", f.Name, terminal),
			Pos:      f.Pos,
		})
	}
}

func (b *builder) resolveReturnProducer(f *model.Flow, sc scope, _ lexer.Position) string {
	wantValue := sc.prefix + f.Return
	if f.ReturnExplicit {
		var fallback string
		for ctrl, inst := range b.instances {
			if inst.ValueLabel != wantValue {
				continue
			}
			if !strings.Contains(ctrl, ".loop.") && !strings.Contains(ctrl, ".repeat.") {
				return ctrl
			}
			if fallback == "" {
				fallback = ctrl
			}
		}
		if fallback != "" {
			return fallback
		}
		return wantValue
	}
	// Rule 0 default: last sequential producer under subflow prefix (exclude subflow value shell).
	prefix := sc.prefix
	var last string
	for ctrl, inst := range b.instances {
		if !strings.HasPrefix(ctrl, prefix) {
			continue
		}
		if inst.ValueLabel == "" {
			continue
		}
		if inst.ReturnsFrom != "" {
			continue
		}
		if last == "" || ctrl > last {
			last = ctrl
		}
	}
	return last
}

func lastSequentialProducer(tree *Node) string {
	if tree == nil {
		return ""
	}
	switch tree.Kind {
	case KindSeq:
		for i := len(tree.Steps) - 1; i >= 0; i-- {
			if p := lastSequentialProducer(tree.Steps[i]); p != "" {
				return p
			}
		}
	case KindStep:
		return tree.Label
	case KindLoop:
		for i := len(tree.Steps) - 1; i >= 0; i-- {
			if p := lastSequentialProducer(tree.Steps[i]); p != "" {
				return p
			}
		}
	case KindParallel:
		if tree.Gather != nil && tree.Gather.Kind == KindStep {
			return tree.Gather.Label
		}
	case KindBranch:
		return "" // ambiguous
	}
	return ""
}

func endsInBranch(tree *Node) bool {
	if tree == nil {
		return false
	}
	if tree.Kind == KindBranch {
		return true
	}
	if tree.Kind == KindSeq && len(tree.Steps) > 0 {
		return endsInBranch(tree.Steps[len(tree.Steps)-1])
	}
	return false
}

func copyMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// wireSequentialEdges sets Upstream on step instances along sequential paths.
func (b *builder) wireSequentialEdges(tree *Node) {
	b.walkSeq(tree, "")
}

func (b *builder) walkSeq(n *Node, upstream string) string {
	if n == nil {
		return upstream
	}
	switch n.Kind {
	case KindSeq:
		prev := upstream
		for _, s := range n.Steps {
			prev = b.walkSeq(s, prev)
		}
		return prev
	case KindStep:
		if inst := b.instances[n.Label]; inst != nil && upstream != "" {
			inst.Upstream = upstream
		}
		if inst := b.instances[n.Label]; inst != nil && inst.ValueLabel != "" {
			return n.Label
		}
		return upstream
	case KindLoop:
		prev := upstream
		for _, s := range n.Steps {
			prev = b.walkSeq(s, prev)
		}
		return prev
	case KindParallel:
		prev := upstream
		for _, s := range n.Steps {
			_ = b.walkSeq(s, prev)
		}
		if n.Gather != nil {
			return b.walkSeq(n.Gather, prev)
		}
		return prev
	case KindBranch:
		for _, c := range n.Cases {
			_ = b.walkSeq(c.Step, upstream)
		}
		return upstream
	default:
		return upstream
	}
}

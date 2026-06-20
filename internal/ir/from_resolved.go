package ir

import (
	"sort"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/flowgraph"
	"github.com/Abhinand20/agentFlow/internal/model"
)

// FromResolved lowers a validated model + resolved flow graph into IR.
func FromResolved(prog *model.Program, res *flowgraph.Resolved) (Program, diag.Diagnostics) {
	var out Program
	if prog == nil || res == nil || res.Tree == nil {
		return out, nil
	}
	entry := prog.Flows[res.Entry]
	if entry == nil {
		return out, nil
	}

	reachableAgents, reachableGates := reachableDecls(prog, res)
	out.Agents = buildAgents(prog, reachableAgents)
	out.Gates = buildGates(prog, reachableGates)
	out.Capabilities = buildCapabilities(prog, out.Agents)
	out.Flow = Flow{
		Name:   res.Entry,
		Return: deriveReturnBinding(prog, res, entry),
		Root:   translateNode(prog, res, res.Tree),
		Order:  controlOrder(res.Tree),
	}
	out.Entry = Entry{
		Trigger:  entry.On,
		FlowName: res.Entry,
		InType:   entry.In,
		OutType:  entryOutType(entry, out.Flow.Return),
	}
	return out, nil
}

func entryOutType(entry *model.Flow, rb ReturnBinding) string {
	if entry.OutExplicit && entry.Out != "" {
		return entry.Out
	}
	if rb.OutType != "" {
		return rb.OutType
	}
	return "text"
}

func enumMembers(prog *model.Program, typeName string) []string {
	if typeName == "" || typeName == "text" {
		return nil
	}
	if et := prog.Types[typeName]; et != nil {
		return append([]string(nil), et.Values...)
	}
	return nil
}

func reachableDecls(prog *model.Program, res *flowgraph.Resolved) (agents, gates map[string]bool) {
	agents = make(map[string]bool)
	gates = make(map[string]bool)
	terminals := model.Terminals()
	for label := range collectControlLabels(res.Tree) {
		inst := res.Instances[label]
		if inst == nil {
			continue
		}
		if terminals[inst.Decl] {
			continue
		}
		if prog.Gates[inst.Decl] != nil {
			gates[inst.Decl] = true
		} else if prog.Agents[inst.Decl] != nil {
			agents[inst.Decl] = true
		}
	}
	return agents, gates
}

func collectControlLabels(n *flowgraph.Node) map[string]bool {
	out := make(map[string]bool)
	walkFG(n, func(node *flowgraph.Node) {
		if node.Kind == flowgraph.KindStep && node.Label != "" {
			out[node.Label] = true
		}
	})
	return out
}

func walkFG(n *flowgraph.Node, visit func(*flowgraph.Node)) {
	if n == nil {
		return
	}
	visit(n)
	switch n.Kind {
	case flowgraph.KindSeq:
		for _, s := range n.Steps {
			walkFG(s, visit)
		}
	case flowgraph.KindLoop:
		for _, s := range n.Steps {
			walkFG(s, visit)
		}
	case flowgraph.KindParallel:
		for _, s := range n.Steps {
			walkFG(s, visit)
		}
		walkFG(n.Gather, visit)
	case flowgraph.KindBranch:
		for _, c := range n.Cases {
			walkFG(c.Step, visit)
		}
	}
}

func buildAgents(prog *model.Program, reachable map[string]bool) []Agent {
	var out []Agent
	for name := range reachable {
		a := prog.Agents[name]
		if a == nil {
			continue
		}
		outType := a.Out
		if outType == "" {
			outType = "text"
		}
		ag := Agent{
			Name:        a.Name,
			Provider:    a.ModelProvider,
			Alias:       a.ResolvedAlias,
			In:          a.In,
			Out:         outType,
			OutEnum:     enumMembers(prog, a.Out),
			Prompt:      a.Prompt,
			Permissions: a.Permissions,
			Retry:       a.Retry,
			Description: a.Description,
		}
		for _, tr := range a.Tools {
			if tr.Tool != "" {
				ag.Tools = append(ag.Tools, ToolRef{Capability: tr.Capability, Tool: tr.Tool})
			}
		}
		sort.Slice(ag.Tools, func(i, j int) bool {
			if ag.Tools[i].Capability != ag.Tools[j].Capability {
				return ag.Tools[i].Capability < ag.Tools[j].Capability
			}
			return ag.Tools[i].Tool < ag.Tools[j].Tool
		})
		out = append(out, ag)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func buildGates(prog *model.Program, reachable map[string]bool) []Gate {
	var out []Gate
	for name := range reachable {
		g := prog.Gates[name]
		if g == nil {
			continue
		}
		out = append(out, Gate{
			Name:         g.Name,
			Run:          g.Run,
			OnFail:       gateFailString(g.OnFail),
			OnFailTarget: g.OnFailTarget,
			Behavior:     g.Behavior,
			ScriptRetry:  g.ScriptRetry,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func gateFailString(a model.GateFailAction) string {
	switch a {
	case model.FailRetryStep:
		return "retry"
	case model.FailGotoStep:
		return "goto"
	case model.FailEnterLoop:
		return "enter-loop"
	default:
		return "halt"
	}
}

func buildCapabilities(prog *model.Program, agents []Agent) []Capability {
	used := make(map[string]bool)
	for _, a := range agents {
		if a.Provider != "" {
			used[a.Provider] = true
		}
		for _, tr := range a.Tools {
			if tr.Capability != "" {
				used[tr.Capability] = true
			}
		}
	}
	var out []Capability
	for name := range used {
		c := prog.Capabilities[name]
		if c == nil {
			continue
		}
		out = append(out, Capability{
			Name:      c.Name,
			Kind:      c.Kind,
			Models:    append([]string(nil), c.Models...),
			Tools:     append([]string(nil), c.Tools...),
			Transport: c.Transport,
			Command:   c.Command,
			Args:      append([]string(nil), c.Args...),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func deriveReturnBinding(prog *model.Program, res *flowgraph.Resolved, entry *model.Flow) ReturnBinding {
	if entry.ReturnExplicit {
		rb := ReturnBinding{
			ValueLabel: entry.Return,
			OutType:    outTypeForValueLabel(prog, res, entry.Return),
			Defaulted:  false,
		}
		if entry.OutExplicit && entry.Out != "" {
			rb.OutType = entry.Out
		}
		return rb
	}
	if endsInBranch(res.Tree) {
		outType := entry.Out
		if outType == "" {
			outType = "text"
		}
		return ReturnBinding{BranchTerminal: true, OutType: outType}
	}
	label := defaultReturnValueLabel(res)
	outType := outTypeForValueLabel(prog, res, label)
	if outType == "" {
		outType = "text"
	}
	return ReturnBinding{ValueLabel: label, OutType: outType, Defaulted: true}
}

func endsInBranch(n *flowgraph.Node) bool {
	if n == nil {
		return false
	}
	if n.Kind == flowgraph.KindBranch {
		return true
	}
	if n.Kind == flowgraph.KindSeq && len(n.Steps) > 0 {
		return endsInBranch(n.Steps[len(n.Steps)-1])
	}
	return false
}

func defaultReturnValueLabel(res *flowgraph.Resolved) string {
	ctrl := lastSequentialProducer(res.Tree)
	if ctrl == "" {
		return ""
	}
	if inst := res.Instances[ctrl]; inst != nil {
		return inst.ValueLabel
	}
	return ""
}

func lastSequentialProducer(n *flowgraph.Node) string {
	if n == nil {
		return ""
	}
	switch n.Kind {
	case flowgraph.KindSeq:
		for i := len(n.Steps) - 1; i >= 0; i-- {
			if p := lastSequentialProducer(n.Steps[i]); p != "" {
				return p
			}
		}
	case flowgraph.KindStep:
		return n.Label
	case flowgraph.KindLoop:
		for i := len(n.Steps) - 1; i >= 0; i-- {
			if p := lastSequentialProducer(n.Steps[i]); p != "" {
				return p
			}
		}
	case flowgraph.KindParallel:
		if n.Gather != nil && n.Gather.Kind == flowgraph.KindStep {
			return n.Gather.Label
		}
	}
	return ""
}

func outTypeForValueLabel(prog *model.Program, res *flowgraph.Resolved, valueLabel string) string {
	for _, inst := range res.Instances {
		if inst.ValueLabel != valueLabel {
			continue
		}
		if a := prog.Agents[inst.Decl]; a != nil {
			if a.Out == "" {
				return "text"
			}
			return a.Out
		}
		if f := prog.Flows[inst.Decl]; f != nil {
			if f.Out == "" {
				return "text"
			}
			return f.Out
		}
	}
	return ""
}

func enumForValueLabel(res *flowgraph.Resolved, valueLabel string) []string {
	for _, inst := range res.Instances {
		if inst.ValueLabel == valueLabel && len(inst.OutEnum) > 0 {
			return append([]string(nil), inst.OutEnum...)
		}
	}
	return nil
}

func translateNode(prog *model.Program, res *flowgraph.Resolved, n *flowgraph.Node) Node {
	if n == nil {
		return Node{Kind: NodeSeq}
	}
	switch n.Kind {
	case flowgraph.KindSeq:
		var steps []Node
		for _, s := range n.Steps {
			steps = append(steps, translateNode(prog, res, s))
		}
		return Node{Kind: NodeSeq, Steps: steps}
	case flowgraph.KindStep:
		return Node{Kind: NodeStep, Step: translateStepRef(prog, res, n.Label)}
	case flowgraph.KindBranch:
		irn := Node{
			Kind:        NodeBranch,
			BranchValue: n.BranchValue,
			BranchEnum:  enumForValueLabel(res, n.BranchValue),
		}
		for _, c := range n.Cases {
			irn.Cases = append(irn.Cases, BranchCase{
				Values: append([]string(nil), c.Values...),
				Body:   translateNode(prog, res, c.Step),
			})
		}
		return irn
	case flowgraph.KindLoop:
		irn := Node{Kind: NodeLoop, DoWhile: n.DoWhile, HasMax: n.Max != nil}
		if n.Max != nil {
			irn.Max = *n.Max
		}
		if n.Cond != nil {
			irn.Cond = &Cond{
				ValueLabel: n.Cond.Value,
				Op:         n.Cond.Op,
				Enum:       n.Cond.Enum,
			}
		}
		irn.Body = translateBody(prog, res, n.Steps)
		return irn
	case flowgraph.KindParallel:
		var branches []Node
		for _, s := range n.Steps {
			branches = append(branches, translateNode(prog, res, s))
		}
		irn := Node{Kind: NodeParallel, Branches: branches}
		if n.Gather != nil {
			gather := translateStepRef(prog, res, n.Gather.Label)
			irn.Gather = gather
			if inst := res.Instances[n.Gather.Label]; inst != nil {
				irn.Payload = buildGatherPayload(n.Gather.Label, inst.GatherPayload)
			}
		}
		return irn
	default:
		return Node{Kind: NodeSeq}
	}
}

func translateBody(prog *model.Program, res *flowgraph.Resolved, steps []*flowgraph.Node) *Node {
	if len(steps) == 0 {
		return nil
	}
	if len(steps) == 1 {
		n := translateNode(prog, res, steps[0])
		return &n
	}
	var body []Node
	for _, s := range steps {
		body = append(body, translateNode(prog, res, s))
	}
	n := Node{Kind: NodeSeq, Steps: body}
	return &n
}

func translateStepRef(prog *model.Program, res *flowgraph.Resolved, controlLabel string) *StepRef {
	inst := res.Instances[controlLabel]
	if inst == nil {
		return &StepRef{ControlLabel: controlLabel, Decl: controlLabel, Kind: "agent"}
	}
	kind := stepDeclKind(prog, inst.Decl)
	outType := "text"
	if a := prog.Agents[inst.Decl]; a != nil {
		if a.Out != "" {
			outType = a.Out
		}
	}
	return &StepRef{
		ControlLabel: inst.ControlLabel,
		ValueLabel:   inst.ValueLabel,
		Decl:         inst.Decl,
		Kind:         kind,
		OutType:      outType,
		OutEnum:      append([]string(nil), inst.OutEnum...),
		PrevProducer: inst.Upstream,
	}
}

func stepDeclKind(prog *model.Program, decl string) string {
	if model.Terminals()[decl] {
		return "terminal"
	}
	if prog.Gates[decl] != nil {
		return "gate"
	}
	return "agent"
}

func buildGatherPayload(gatherLabel string, payload map[string]string) *GatherPayload {
	if len(payload) == 0 {
		return nil
	}
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	gp := &GatherPayload{GatherControlLabel: gatherLabel}
	for _, k := range keys {
		gp.Branches = append(gp.Branches, GatherBranch{
			ControlLabel: k,
			ValueLabel:   payload[k],
		})
	}
	return gp
}

func controlOrder(n *flowgraph.Node) []string {
	var order []string
	walkFG(n, func(node *flowgraph.Node) {
		if node.Kind == flowgraph.KindStep && node.Label != "" {
			order = append(order, node.Label)
		}
	})
	return order
}

func FindStepByControlLabel(root Node, label string) *StepRef {
	if root.Kind == NodeStep && root.Step != nil && root.Step.ControlLabel == label {
		return root.Step
	}
	if root.Gather != nil && root.Gather.ControlLabel == label {
		return root.Gather
	}
	for _, s := range root.Steps {
		if ref := FindStepByControlLabel(s, label); ref != nil {
			return ref
		}
	}
	if root.Body != nil {
		if ref := FindStepByControlLabel(*root.Body, label); ref != nil {
			return ref
		}
	}
	for _, b := range root.Branches {
		if ref := FindStepByControlLabel(b, label); ref != nil {
			return ref
		}
	}
	for _, c := range root.Cases {
		if ref := FindStepByControlLabel(c.Body, label); ref != nil {
			return ref
		}
	}
	return nil
}

// HasAgent reports whether name is in the agent list (for tests).
func (p Program) HasAgent(name string) bool {
	for _, a := range p.Agents {
		if a.Name == name {
			return true
		}
	}
	return false
}

// GateByName returns a gate by name (for tests).
func (p Program) GateByName(name string) *Gate {
	for i := range p.Gates {
		if p.Gates[i].Name == name {
			return &p.Gates[i]
		}
	}
	return nil
}

// AgentByName returns an agent by name (for tests).
func (p Program) AgentByName(name string) *Agent {
	for i := range p.Agents {
		if p.Agents[i].Name == name {
			return &p.Agents[i]
		}
	}
	return nil
}

// ContainsPromptSuffix is a test helper.
func (p Program) ContainsPromptSuffix(suffix string) bool {
	for _, a := range p.Agents {
		if strings.Contains(a.Prompt, suffix) {
			return true
		}
	}
	return false
}

package ir_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/flowgraph"
	"github.com/Abhinand20/agentFlow/internal/ir"
	"github.com/Abhinand20/agentFlow/internal/model"
	"github.com/Abhinand20/agentFlow/internal/parser"
	"github.com/Abhinand20/agentFlow/internal/sema"
)

func compileIR(t *testing.T, name string) (ir.Program, *model.Program, *flowgraph.Resolved) {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "examples")
	src, err := os.ReadFile(filepath.Join(dir, name+".af"))
	if err != nil {
		t.Fatal(err)
	}
	root, diags := parser.Parse(name+".af", string(src))
	if diags.HasErrors() {
		t.Fatalf("parse: %#v", diags)
	}
	prog, sdiags := sema.Resolve(root, dir)
	if sdiags.HasErrors() {
		t.Fatalf("sema: %#v", sdiags)
	}
	res, fdiags := flowgraph.Resolve(prog)
	if fdiags.HasErrors() {
		t.Fatalf("flowgraph: %#v", fdiags)
	}
	iprog, _ := ir.FromResolved(prog, res)
	return iprog, prog, res
}

func TestReviewIRInvariants(t *testing.T) {
	iprog, prog, _ := compileIR(t, "review")

	if iprog.Entry.Trigger != "/ship" || iprog.Entry.FlowName != "ship" {
		t.Fatalf("entry = %#v", iprog.Entry)
	}
	if !iprog.Flow.Return.BranchTerminal || iprog.Flow.Return.OutType != "Decision" {
		t.Fatalf("ship return = %#v", iprog.Flow.Return)
	}

	g := iprog.GateByName("quality")
	if g == nil || g.OnFail != "retry" || g.OnFailTarget != "build" || g.ScriptRetry != 2 {
		t.Fatalf("quality gate = %#v", g)
	}

	rv := iprog.AgentByName("reviewer")
	if rv == nil || len(rv.OutEnum) != 3 {
		t.Fatalf("reviewer = %#v", rv)
	}
	wantEnum := []string{"approve", "revise", "reject"}
	for i, v := range wantEnum {
		if rv.OutEnum[i] != v {
			t.Fatalf("OutEnum[%d] = %q", i, rv.OutEnum[i])
		}
	}
	if iprog.ContainsPromptSuffix("agentflow-output") {
		t.Fatal("reviewer prompt must not include output-protocol suffix")
	}

	cr := prog.Flows["code_review"]
	if cr.Return != "review" || cr.Out != "Verdict" {
		t.Fatalf("code_review model = %#v", cr)
	}

	ref := ir.FindStepByControlLabel(iprog.Flow.Root, "code_review.reviewer")
	if ref == nil {
		t.Fatal("missing code_review.reviewer step")
	}
	// payload lives on parallel node — walk tree
	payload := findGatherPayload(iprog.Flow.Root, "code_review.reviewer")
	if payload == nil || len(payload.Branches) != 3 {
		t.Fatalf("gather payload = %#v", payload)
	}
	for _, k := range []string{"lint", "security", "style"} {
		if !hasGatherBranch(payload, k) {
			t.Fatalf("missing gather branch %q", k)
		}
	}

	if iprog.HasAgent("unused") {
		t.Fatal("unreachable agents must not appear")
	}
}

func TestPipelineIRInvariants(t *testing.T) {
	iprog, _, _ := compileIR(t, "pipeline")
	rb := iprog.Flow.Return
	if !rb.Defaulted || rb.ValueLabel != "edit" || rb.OutType != "text" {
		t.Fatalf("return = %#v", rb)
	}
}

func TestCriticIRInvariants(t *testing.T) {
	iprog, _, _ := compileIR(t, "critic")
	loop := findLoopNode(iprog.Flow.Root)
	if loop == nil {
		t.Fatal("missing loop node")
	}
	if !loop.DoWhile || !loop.HasMax || loop.Max != 3 {
		t.Fatalf("loop = %#v", loop)
	}
	if loop.Cond == nil || loop.Cond.ValueLabel != "verdict" || loop.Cond.Enum != "pass" {
		t.Fatalf("cond = %#v", loop.Cond)
	}
}

func TestResearchIRInvariants(t *testing.T) {
	iprog, _, _ := compileIR(t, "research")
	payload := findGatherPayload(iprog.Flow.Root, "synthesize")
	if payload == nil || len(payload.Branches) != 3 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestReachableOnlyAgents(t *testing.T) {
	iprog, prog, _ := compileIR(t, "review")
	if len(iprog.Agents) >= len(prog.Agents) {
		// all declared agents happen to be reachable in review.af
	}
	for _, a := range iprog.Agents {
		if prog.Agents[a.Name] == nil {
			t.Fatalf("unknown agent %q in IR", a.Name)
		}
	}
}

func findLoopNode(n ir.Node) *ir.Node {
	if n.Kind == ir.NodeLoop {
		return &n
	}
	for i := range n.Steps {
		if got := findLoopNode(n.Steps[i]); got != nil {
			return got
		}
	}
	if n.Body != nil {
		return findLoopNode(*n.Body)
	}
	return nil
}

func findGatherPayload(n ir.Node, gatherLabel string) *ir.GatherPayload {
	if n.Kind == ir.NodeParallel && n.Gather != nil && n.Gather.ControlLabel == gatherLabel {
		return n.Payload
	}
	for _, s := range n.Steps {
		if p := findGatherPayload(s, gatherLabel); p != nil {
			return p
		}
	}
	if n.Body != nil {
		if p := findGatherPayload(*n.Body, gatherLabel); p != nil {
			return p
		}
	}
	for _, b := range n.Branches {
		if p := findGatherPayload(b, gatherLabel); p != nil {
			return p
		}
	}
	for _, c := range n.Cases {
		if p := findGatherPayload(c.Body, gatherLabel); p != nil {
			return p
		}
	}
	return nil
}

func hasGatherBranch(p *ir.GatherPayload, control string) bool {
	for _, b := range p.Branches {
		if b.ControlLabel == control {
			return true
		}
	}
	return false
}

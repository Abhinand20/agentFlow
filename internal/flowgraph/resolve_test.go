package flowgraph

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
	"github.com/Abhinand20/agentFlow/internal/parser"
	"github.com/Abhinand20/agentFlow/internal/sema"
)

func examplesDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "examples")
}

func resolveFixture(t *testing.T, name string) (*Resolved, *model.Program) {
	t.Helper()
	dir := examplesDir(t)
	src, err := os.ReadFile(filepath.Join(dir, name+".af"))
	if err != nil {
		t.Fatal(err)
	}
	root, diags := parser.Parse(name+".af", string(src))
	if diags.HasErrors() {
		t.Fatalf("parse: %#v", diags)
	}
	prog, _ := sema.Resolve(root, dir)
	res, _ := Resolve(prog)
	return res, prog
}

func TestShipTopLevelShape(t *testing.T) {
	res, _ := resolveFixture(t, "review")
	if res.Entry != "ship" {
		t.Fatalf("entry = %q", res.Entry)
	}
	if res.Tree == nil || res.Tree.Kind != KindSeq {
		t.Fatalf("tree = %#v", res.Tree)
	}
	steps := res.Tree.Steps
	if len(steps) < 2 {
		t.Fatalf("expected at least 2 top-level steps, got %d", len(steps))
	}
	last := steps[len(steps)-1]
	if last.Kind != KindBranch || last.BranchValue != "code_review" {
		t.Fatalf("last step = %#v", last)
	}
}

func TestDocsChainShape(t *testing.T) {
	res, _ := resolveFixture(t, "docs")
	if res.Tree == nil || res.Tree.Kind != KindSeq {
		t.Fatalf("tree = %#v", res.Tree)
	}
	if len(res.Tree.Steps) != 2 {
		t.Fatalf("steps = %d", len(res.Tree.Steps))
	}
	if res.Tree.Steps[0].Label != "outline" || res.Tree.Steps[1].Label != "draft" {
		t.Fatalf("labels = %q, %q", res.Tree.Steps[0].Label, res.Tree.Steps[1].Label)
	}
}

func TestLabelAssignmentDefaultAndAlias(t *testing.T) {
	prog := &model.Program{
		Types: map[string]*model.EnumType{
			"Verdict": {Name: "Verdict", Values: []string{"approve", "revise", "reject"}},
		},
		Agents: map[string]*model.Agent{
			"reviewer": {Name: "reviewer", Out: "Verdict"},
		},
		Flows: map[string]*model.Flow{
			"main": {
				Name: "main", Entry: true,
				Body: []model.Step{
					{Kind: model.StepRef, Ref: "reviewer", Alias: "review"},
				},
			},
		},
		EntryFlow: "main",
	}
	res, _ := Resolve(prog)
	inst := res.Instances["reviewer"]
	if inst == nil {
		t.Fatal("missing reviewer instance")
	}
	if inst.ControlLabel != "reviewer" || inst.ValueLabel != "review" {
		t.Fatalf("inst = %#v", inst)
	}
}

func TestLabelDisambiguation(t *testing.T) {
	prog := &model.Program{
		Agents: map[string]*model.Agent{
			"build": {Name: "build"},
		},
		Flows: map[string]*model.Flow{
			"main": {
				Name: "main", Entry: true,
				Body: []model.Step{
					{Kind: model.StepRef, Ref: "build"},
					{Kind: model.StepRef, Ref: "build"},
				},
			},
		},
		EntryFlow: "main",
	}
	res, _ := Resolve(prog)
	if res.Instances["build"] == nil || res.Instances["build#2"] == nil {
		t.Fatalf("instances = %#v", res.Instances)
	}
}

func TestSubflowInliningLabels(t *testing.T) {
	res, _ := resolveFixture(t, "review")
	if res.Instances["code_review.build"] == nil {
		t.Fatal("missing code_review.build")
	}
	if res.Instances["code_review.quality"] == nil {
		t.Fatal("missing code_review.quality")
	}
	cr := res.Instances["code_review"]
	if cr == nil {
		t.Fatal("missing code_review subflow value")
	}
	if cr.ReturnsFrom != "code_review.reviewer" {
		t.Fatalf("ReturnsFrom = %q", cr.ReturnsFrom)
	}
	if len(cr.OutEnum) != 3 {
		t.Fatalf("OutEnum = %#v", cr.OutEnum)
	}
}

func TestRecursiveFlowCycle(t *testing.T) {
	pos := lexer.Position{Line: 1}
	prog := &model.Program{
		Flows: map[string]*model.Flow{
			"a": {
				Name: "a", Entry: true, Pos: pos,
				Body: []model.Step{{Kind: model.StepRef, Ref: "b", Pos: pos}},
			},
			"b": {
				Name: "b", Pos: pos,
				Body: []model.Step{{Kind: model.StepRef, Ref: "a", Pos: pos}},
			},
		},
		EntryFlow: "a",
	}
	_, diags := Resolve(prog)
	if !diags.HasErrors() {
		t.Fatal("expected cycle error")
	}
	found := false
	for _, d := range diags {
		if d.Code == "AF212" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestRepeatNormalization(t *testing.T) {
	res, _ := resolveFixture(t, "critic")
	if len(res.Tree.Steps) != 1 || res.Tree.Steps[0].Kind != KindLoop {
		t.Fatalf("tree = %#v", res.Tree)
	}
	loop := res.Tree.Steps[0]
	if !loop.DoWhile {
		t.Fatal("expected do-while repeat")
	}
	if loop.Cond == nil || loop.Cond.Value != "verdict" || loop.Cond.Enum != "pass" {
		t.Fatalf("cond = %#v", loop.Cond)
	}
	if loop.Max == nil || *loop.Max != 3 {
		t.Fatalf("max = %#v", loop.Max)
	}
	if res.Instances["repeat.generate"] == nil || res.Instances["repeat.critic"] == nil {
		t.Fatalf("instances = %#v", res.Instances)
	}
	if res.Instances["repeat.critic"].ValueLabel != "verdict" {
		t.Fatalf("critic value = %q", res.Instances["repeat.critic"].ValueLabel)
	}
}

func TestGatherPayloadReview(t *testing.T) {
	res, _ := resolveFixture(t, "review")
	rev := res.Instances["code_review.reviewer"]
	if rev == nil {
		t.Fatal("missing gather step")
	}
	want := map[string]string{
		"lint":     "lint",
		"security": "security",
		"style":    "style",
	}
	if len(rev.GatherPayload) != 3 {
		t.Fatalf("payload = %#v", rev.GatherPayload)
	}
	for k, v := range want {
		if rev.GatherPayload[k] != v {
			t.Fatalf("payload[%q] = %q, want %q", k, rev.GatherPayload[k], v)
		}
	}
}

func TestGatherPayloadResearch(t *testing.T) {
	res, _ := resolveFixture(t, "research")
	syn := res.Instances["synthesize"]
	if syn == nil {
		t.Fatal("missing synthesize")
	}
	for _, k := range []string{"market", "competitor", "financial"} {
		if syn.GatherPayload[k] != k {
			t.Fatalf("payload[%q] = %q", k, syn.GatherPayload[k])
		}
	}
}

func TestDefaultReturnPipeline(t *testing.T) {
	res, _ := resolveFixture(t, "pipeline")
	inst := res.Instances["edit"]
	if inst == nil {
		t.Fatal("missing edit")
	}
	if inst.ValueLabel != "edit" {
		t.Fatalf("value = %q", inst.ValueLabel)
	}
}

func TestDefaultReturnResearch(t *testing.T) {
	res, _ := resolveFixture(t, "research")
	syn := res.Instances["synthesize"]
	if syn == nil || syn.ValueLabel != "report" {
		t.Fatalf("synthesize = %#v", syn)
	}
}

func TestExplicitReturnCritic(t *testing.T) {
	res, _ := resolveFixture(t, "critic")
	gen := res.Instances["repeat.generate"]
	if gen == nil || gen.ValueLabel != "draft" {
		t.Fatalf("generate = %#v", gen)
	}
}

func TestGateTerminalAF209(t *testing.T) {
	pos := lexer.Position{Line: 1}
	prog := &model.Program{
		Gates: map[string]*model.Gate{
			"quality": {Name: "quality"},
		},
		Flows: map[string]*model.Flow{
			"main": {
				Name: "main", Entry: true, Pos: pos,
				Body: []model.Step{{Kind: model.StepRef, Ref: "quality", Pos: pos}},
			},
		},
		EntryFlow: "main",
	}
	_, diags := Resolve(prog)
	if !diags.HasErrors() {
		t.Fatal("expected AF209")
	}
	found := false
	for _, d := range diags {
		if d.Code == "AF209" && d.Severity == diag.Error {
			found = true
		}
	}
	if !found {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestSequentialUpstream(t *testing.T) {
	res, _ := resolveFixture(t, "review")
	q := res.Instances["code_review.quality"]
	if q == nil || q.Upstream != "code_review.build" {
		t.Fatalf("quality upstream = %#v", q)
	}
}

func TestOutEnumReviewer(t *testing.T) {
	res, _ := resolveFixture(t, "review")
	// parallel gather reviewer (outside loop)
	rev := res.Instances["code_review.reviewer"]
	if rev == nil {
		t.Fatal("missing reviewer")
	}
	want := []string{"approve", "revise", "reject"}
	if len(rev.OutEnum) != len(want) {
		t.Fatalf("OutEnum = %#v", rev.OutEnum)
	}
	for i, v := range want {
		if rev.OutEnum[i] != v {
			t.Fatalf("OutEnum[%d] = %q", i, rev.OutEnum[i])
		}
	}
}

func TestBranchArmDedup(t *testing.T) {
	res, _ := resolveFixture(t, "review")
	branch := res.Tree.Steps[len(res.Tree.Steps)-1]
	if branch.Kind != KindBranch {
		t.Fatal("expected branch")
	}
	revise := branch.Cases[1].Step
	reject := branch.Cases[2].Step
	if revise.Label != reject.Label {
		t.Fatalf("revise=%q reject=%q", revise.Label, reject.Label)
	}
	if res.Instances["notify_author"] == nil {
		t.Fatal("expected single notify_author instance")
	}
}

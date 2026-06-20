package sema

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/model"
	"github.com/Abhinand20/agentFlow/internal/parser"
)

var update = flag.Bool("update", false, "rewrite golden model snapshots")

func TestGoldenModel(t *testing.T) {
	for _, name := range []string{"review", "pipeline", "research", "critic", "docs"} {
		t.Run(name, func(t *testing.T) {
			_, file, _, _ := runtime.Caller(0)
			examplesDir := filepath.Join(filepath.Dir(file), "..", "..", "examples")
			src, err := os.ReadFile(filepath.Join(examplesDir, name+".af"))
			if err != nil {
				t.Fatal(err)
			}
			root, diags := parser.Parse(name+".af", string(src))
			if diags.HasErrors() {
				t.Fatalf("parse: %#v", diags)
			}
			prog, _ := Resolve(root, examplesDir)
			got, err := model.MarshalSnapshot(prog)
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", name+".model.json")
			if *update {
				os.MkdirAll("testdata", 0o755)
				os.WriteFile(golden, got, 0o644)
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run -update): %v", err)
			}
			if string(got) != string(want) {
				t.Fatalf("snapshot mismatch for %s", name)
			}
		})
	}
}

func TestReviewModelInvariants(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "examples")
	src, _ := os.ReadFile(filepath.Join(dir, "review.af"))
	root, _ := parser.Parse("review.af", string(src))
	prog, _ := Resolve(root, dir)

	rv := prog.Agents["reviewer"]
	pf, _ := os.ReadFile(filepath.Join(dir, "prompts", "reviewer.md"))
	if rv.Out != "Verdict" || !rv.PromptFromFile || rv.Prompt != string(pf) {
		t.Fatalf("reviewer = %#v", rv)
	}
	if prog.Agents["build"].ModelProvider != "anthropic" || prog.Agents["build"].ResolvedAlias != "sonnet" {
		t.Fatalf("build model = %#v", prog.Agents["build"])
	}
	q := prog.Gates["quality"]
	if q.OnFail != model.FailRetryStep || q.OnFailTarget != "build" || q.Behavior != "blocking" || q.ScriptRetry != 2 {
		t.Fatalf("gate = %#v", q)
	}
	cr := prog.Flows["code_review"]
	if cr.Return != "review" || !cr.ReturnExplicit || cr.Out != "Verdict" || !cr.OutExplicit {
		t.Fatalf("code_review = %#v", cr)
	}
	sh := prog.Flows["ship"]
	if !sh.Entry || sh.On != "/ship" || sh.Return != "" || sh.ReturnExplicit {
		t.Fatalf("ship = %#v", sh)
	}
}

func TestPipelineModelInvariants(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "examples")
	src, _ := os.ReadFile(filepath.Join(dir, "pipeline.af"))
	root, _ := parser.Parse("pipeline.af", string(src))
	prog, _ := Resolve(root, dir)
	c := prog.Flows["content"]
	if c.OutExplicit || c.ReturnExplicit {
		t.Fatalf("content = %#v", c)
	}
}

func TestCriticModelInvariants(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "examples")
	src, _ := os.ReadFile(filepath.Join(dir, "critic.af"))
	root, _ := parser.Parse("critic.af", string(src))
	prog, _ := Resolve(root, dir)
	sql := prog.Flows["sql"]
	if sql.Return != "draft" || !sql.ReturnExplicit || sql.Out != "text" {
		t.Fatalf("sql = %#v", sql)
	}
	st := sql.Body[0]
	if st.Kind != model.StepRepeat || !st.DoWhile || st.Cond == nil || st.Cond.Value != "verdict" || st.Cond.Enum != "pass" {
		t.Fatalf("repeat = %#v", st)
	}
	if st.Max == nil || *st.Max != 3 {
		t.Fatalf("max = %#v", st.Max)
	}
}

func TestDocsModelInvariants(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "examples")
	src, _ := os.ReadFile(filepath.Join(dir, "docs.af"))
	root, _ := parser.Parse("docs.af", string(src))
	prog, _ := Resolve(root, dir)
	outlinePF, _ := os.ReadFile(filepath.Join(dir, "prompts", "outline.md"))
	draftPF, _ := os.ReadFile(filepath.Join(dir, "prompts", "draft.md"))
	o := prog.Agents["outline"]
	if o.Prompt != string(outlinePF) || !o.PromptFromFile || o.PromptPath != "prompts/outline.md" {
		t.Fatalf("outline = %#v", o)
	}
	d := prog.Agents["draft"]
	if d.Prompt != string(draftPF) || !d.PromptFromFile {
		t.Fatalf("draft = %#v", d)
	}
}

func TestResearchModelInvariants(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "examples")
	src, _ := os.ReadFile(filepath.Join(dir, "research.af"))
	root, _ := parser.Parse("research.af", string(src))
	prog, _ := Resolve(root, dir)
	if prog.Agents["synthesize"].Out != "Report" {
		t.Fatalf("synthesize = %#v", prog.Agents["synthesize"])
	}
	par := prog.Flows["research"].Body[0]
	if par.Kind != model.StepParallel || par.Gather == nil || par.Gather.Ref != "synthesize" || par.Gather.Alias != "report" {
		t.Fatalf("parallel = %#v", par)
	}
}

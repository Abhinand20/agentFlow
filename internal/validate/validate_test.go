package validate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/flowgraph"
	"github.com/Abhinand20/agentFlow/internal/model"
	"github.com/Abhinand20/agentFlow/internal/parser"
	"github.com/Abhinand20/agentFlow/internal/sema"
	"github.com/Abhinand20/agentFlow/internal/validate"
)

func compile(t *testing.T, src, srcDir string) (prog *model.Program, res *flowgraph.Resolved, prior diag.Diagnostics) {
	t.Helper()
	if srcDir == "" {
		srcDir = "."
	}
	root, pdiags := parser.Parse("test.af", src)
	if pdiags.HasErrors() {
		t.Fatalf("parse failed: %#v", pdiags)
	}
	prog, rdiags := sema.Resolve(root, srcDir)
	res, fdiags := flowgraph.Resolve(prog)
	var all diag.Diagnostics
	all.Add(rdiags...)
	all.Add(fdiags...)
	return prog, res, all
}

func check(t *testing.T, src, srcDir string) diag.Diagnostics {
	t.Helper()
	prog, res, prior := compile(t, src, srcDir)
	vdiags := validate.Validate(prog, res)
	var all diag.Diagnostics
	all.Add(prior...)
	all.Add(vdiags...)
	return all
}

func hasCode(diags diag.Diagnostics, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

func assertHasCode(t *testing.T, diags diag.Diagnostics, code string) {
	t.Helper()
	if !hasCode(diags, code) {
		t.Fatalf("expected %s among %#v", code, diags)
	}
}

func assertNoCode(t *testing.T, diags diag.Diagnostics, code string) {
	t.Helper()
	if hasCode(diags, code) {
		t.Fatalf("did not expect %s among %#v", code, diags)
	}
}

func assertOnlyCodes(t *testing.T, diags diag.Diagnostics, codes ...string) {
	t.Helper()
	want := make(map[string]bool, len(codes))
	for _, c := range codes {
		want[c] = true
	}
	for _, d := range diags {
		if d.Severity != diag.Error {
			continue
		}
		if !want[d.Code] {
			t.Fatalf("unexpected error %s: %s", d.Code, d.Msg)
		}
	}
	for _, c := range codes {
		if !hasCode(diags, c) {
			t.Fatalf("missing %s in %#v", c, diags)
		}
	}
}

func examplesDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "examples")
}

func checkExample(t *testing.T, name string) diag.Diagnostics {
	t.Helper()
	dir := examplesDir(t)
	src, err := os.ReadFile(filepath.Join(dir, name+".af"))
	if err != nil {
		t.Fatal(err)
	}
	return check(t, string(src), dir)
}

func validationErrorsOnly(diags diag.Diagnostics) diag.Diagnostics {
	var out diag.Diagnostics
	for _, d := range diags {
		if d.Severity == diag.Error {
			out = append(out, d)
		}
	}
	return out
}

// --- AF200 ---

func TestAF200DuplicateAgent(t *testing.T) {
	src := `agent reviewer { model: opus }
agent reviewer { model: sonnet }
entry flow f { reviewer }`
	assertHasCode(t, check(t, src, ""), "AF200")
}

func TestAF200CrossKind(t *testing.T) {
	src := `agent worker { model: opus }
gate worker { run: "x.sh" }
entry flow f { worker }`
	assertHasCode(t, check(t, src, ""), "AF200")
}

func TestAF200ReservedTerminal(t *testing.T) {
	src := `agent done { model: opus }
entry flow f { done }`
	assertHasCode(t, check(t, src, ""), "AF200")
}

func TestAF200Clean(t *testing.T) {
	src := `agent a { model: opus }
agent b { model: opus }
entry flow f { a -> b }`
	assertNoCode(t, check(t, src, ""), "AF200")
}

// --- AF201 ---

func TestAF201UnknownStep(t *testing.T) {
	src := `agent a { model: opus }
entry flow f { a -> nonexistent }`
	assertHasCode(t, check(t, src, ""), "AF201")
}

func TestAF201Terminals(t *testing.T) {
	src := `type V = ok | bad
agent a { model: opus out: V }
entry flow f { a
  branch a { case ok -> done case bad -> fail } }`
	assertNoCode(t, check(t, src, ""), "AF201")
}

func TestAF201GatherRef(t *testing.T) {
	src := `agent a { model: opus }
entry flow f { parallel { a } gather missing as r }`
	assertHasCode(t, check(t, src, ""), "AF201")
}

// --- AF202 ---

func TestAF202UndeclaredOut(t *testing.T) {
	src := `agent a { model: opus out: Undeclared }
entry flow f { a }`
	assertHasCode(t, check(t, src, ""), "AF202")
}

func TestAF202TextOutOK(t *testing.T) {
	src := `agent a { model: opus out: text }
entry flow f { a }`
	assertNoCode(t, check(t, src, ""), "AF202")
}

func TestAF202OpaqueInOK(t *testing.T) {
	src := `agent a { model: opus in: Ticket }
entry flow f { a }`
	assertNoCode(t, check(t, src, ""), "AF202")
}

// --- AF203 ---

func TestAF203BadCase(t *testing.T) {
	src := `type Verdict = approve | reject
agent reviewer { model: opus out: Verdict }
entry flow f {
  reviewer as review
  branch review { case notamember -> reviewer }
}`
	assertHasCode(t, check(t, src, ""), "AF203")
}

func TestAF203LoopCond(t *testing.T) {
	src := `type V = pass | fail
agent a { model: opus out: V }
entry flow f {
  loop (until verdict == notmember, max 3) { a as verdict }
}`
	assertHasCode(t, check(t, src, ""), "AF203")
}

func TestAF203Clean(t *testing.T) {
	src := `type Verdict = approve | reject
agent reviewer { model: opus out: Verdict }
entry flow f {
  reviewer as review
  branch review { case approve -> reviewer case reject -> reviewer }
}`
	assertNoCode(t, check(t, src, ""), "AF203")
}

// --- AF204 ---

func TestAF204NonExhaustive(t *testing.T) {
	src := `type Verdict = approve | revise | reject
agent reviewer { model: opus out: Verdict }
entry flow f {
  reviewer as review
  branch review { case approve -> reviewer }
}`
	diags := check(t, src, "")
	assertHasCode(t, diags, "AF204")
	for _, d := range diags {
		if d.Code == "AF204" && d.Severity != diag.Warning {
			t.Fatalf("AF204 should be warning: %#v", d)
		}
	}
}

// --- AF205 ---

func TestAF205LoopNoMax(t *testing.T) {
	src := `agent a { model: opus }
entry flow f { loop (until a == done) { a } }`
	assertHasCode(t, check(t, src, ""), "AF205")
}

func TestAF205LoopWithMax(t *testing.T) {
	src := `type V = pass | fail
agent a { model: opus out: V }
entry flow f { loop (until x == pass, max 3) { a as x } }`
	assertNoCode(t, check(t, src, ""), "AF205")
}

// --- AF206 ---

func TestAF206BadTool(t *testing.T) {
	src := `use github { kind: mcp command: "npx" tools: [get_pr] }
agent a { model: opus tools: [github.nope] }
entry flow f { a }`
	assertHasCode(t, check(t, src, ""), "AF206")
}

func TestAF206GoodTool(t *testing.T) {
	src := `use github { kind: mcp command: "npx" tools: [get_pr] }
agent a { model: opus tools: [github.get_pr] }
entry flow f { a }`
	assertNoCode(t, check(t, src, ""), "AF206")
}

// --- AF207 ---

func TestAF207OrphanFlow(t *testing.T) {
	src := `agent a { model: opus }
flow helper { a }
entry flow f { a }`
	diags := check(t, src, "")
	assertHasCode(t, diags, "AF207")
	for _, d := range diags {
		if d.Code == "AF207" && d.Severity != diag.Warning {
			t.Fatalf("AF207 should be warning: %#v", d)
		}
	}
}

// --- AF208 ---

func TestAF208DuplicateBareRef(t *testing.T) {
	src := `agent reviewer { model: opus }
entry flow f { reviewer reviewer }`
	assertHasCode(t, check(t, src, ""), "AF208")
}

func TestAF208AliasOK(t *testing.T) {
	src := `agent reviewer { model: opus }
entry flow f { reviewer as r1 reviewer as r2 }`
	assertNoCode(t, check(t, src, ""), "AF208")
}

// --- AF209 ---

func TestAF209BadReturn(t *testing.T) {
	src := `type V = ok
agent a { model: opus out: V }
flow inner { out: V return: typo a as ok }
entry flow f { inner }`
	assertHasCode(t, check(t, src, ""), "AF209")
}

func TestAF209GateTerminal(t *testing.T) {
	src := `gate quality { run: "x.sh" }
entry flow f { quality }`
	diags := check(t, src, "")
	assertHasCode(t, diags, "AF209") // emitted by M3
}

func TestAF209ExplicitReturnOK(t *testing.T) {
	src := `type Verdict = approve | reject
agent reviewer { model: opus out: Verdict }
flow inner { out: Verdict return: review reviewer as review }
entry flow f { inner }`
	assertNoCode(t, validationErrorsOnly(check(t, src, "")), "AF209")
}

// --- AF210 ---

func TestAF210BranchLeafMismatch(t *testing.T) {
	src := `use anthropic { kind: model-provider models: [opus] }
type Decision = shipped | rejected
type Verdict = approve | reject
agent deploy { model: opus out: Decision }
agent bad { model: opus out: Verdict }
entry flow ship {
  out: Decision
  branch x {
    case a -> deploy
    case b -> bad
  }
}`
	assertHasCode(t, check(t, src, ""), "AF210")
}

func TestAF210BranchLeafMatch(t *testing.T) {
	src := `use anthropic { kind: model-provider models: [opus] }
type Decision = shipped | rejected
agent deploy { model: opus out: Decision }
agent notify { model: opus out: Decision }
entry flow ship {
  out: Decision
  branch x {
    case a -> deploy
    case b -> notify
  }
}`
	assertNoCode(t, validationErrorsOnly(check(t, src, "")), "AF210")
}

// --- AF211 ---

func TestAF211Conflict(t *testing.T) {
	src := `agent a { model: opus prompt: "hi" prompt-file: "x.md" }
entry flow f { a }`
	assertHasCode(t, check(t, src, ""), "AF211")
}

func TestAF211MissingFile(t *testing.T) {
	src := `agent a { model: opus prompt: "missing.md" }
entry flow f { a }`
	assertHasCode(t, check(t, src, ""), "AF211")
}

func TestAF211Escape(t *testing.T) {
	src := `agent a { model: opus prompt-file: "../escape.md" }
entry flow f { a }`
	assertHasCode(t, check(t, src, ""), "AF211")
}

// --- golden programs ---

func TestGoldenReviewClean(t *testing.T) {
	diags := validationErrorsOnly(checkExample(t, "review"))
	if len(diags) > 0 {
		t.Fatalf("review.af errors: %#v", diags)
	}
}

func TestGoldenPipelineDefaultReturn(t *testing.T) {
	diags := validationErrorsOnly(checkExample(t, "pipeline"))
	if len(diags) > 0 {
		t.Fatalf("pipeline.af errors: %#v", diags)
	}
}

func TestGoldenDocsPrompts(t *testing.T) {
	diags := validationErrorsOnly(checkExample(t, "docs"))
	if len(diags) > 0 {
		t.Fatalf("docs.af errors: %#v", diags)
	}
}

func TestGoldenResearchAndCritic(t *testing.T) {
	for _, name := range []string{"research", "critic"} {
		t.Run(name, func(t *testing.T) {
			diags := validationErrorsOnly(checkExample(t, name))
			if len(diags) > 0 {
				t.Fatalf("%s.af errors: %#v", name, diags)
			}
		})
	}
}

// --- aggregation ---

func TestAggregationMultipleErrors(t *testing.T) {
	src := `use anthropic { kind: model-provider models: [opus, sonnet] }
use github { kind: mcp command: "npx" tools: [get_pr] }
agent a { model: opus out: BadType tools: [github.nope] }
agent a { model: sonnet }
entry flow f { a -> missing }`
	prog, res, _ := compile(t, src, "")
	diags := validate.Validate(prog, res)
	assertOnlyCodes(t, diags, "AF200", "AF201", "AF202", "AF206")
}

func TestDeterministicSort(t *testing.T) {
	src := `agent a { model: opus out: BadType }
agent a { model: sonnet }
entry flow f { a -> missing }`
	prog, res, _ := compile(t, src, "")
	d1 := validate.Validate(prog, res)
	d2 := validate.Validate(prog, res)
	if len(d1) == 0 || len(d1) != len(d2) {
		t.Fatalf("expected errors")
	}
	for i := range d1 {
		if d1[i].Code != d2[i].Code || d1[i].Msg != d2[i].Msg {
			t.Fatalf("order unstable at %d: %v vs %v", i, d1[i], d2[i])
		}
	}
}

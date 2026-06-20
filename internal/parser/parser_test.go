package parser

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/ast"
	"github.com/Abhinand20/agentFlow/internal/diag"
)

var update = flag.Bool("update", false, "update golden AST snapshots")

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), "examples", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func parseFixture(t *testing.T, name string) *ast.AST {
	t.Helper()
	path := filepath.Join("examples", name)
	astRoot, diags := Parse(path, readFixture(t, name))
	if diags.HasErrors() {
		t.Fatalf("parse %s: %v", name, diags)
	}
	return astRoot
}

func TestParseEmpty(t *testing.T) {
	astRoot, diags := Parse("empty.af", "")
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	if len(astRoot.Decls) != 0 {
		t.Fatalf("expected 0 decls, got %d", len(astRoot.Decls))
	}
}

func TestParseSemicolonAF000(t *testing.T) {
	_, diags := Parse("bad.af", "agent x { model: sonnet; }")
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}
	if diags[0].Code != "AF000" {
		t.Fatalf("code = %q", diags[0].Code)
	}
}

func TestParseUnclosedBrace(t *testing.T) {
	_, diags := Parse("bad.af", "agent x {")
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}
}

func TestParseUnterminatedString(t *testing.T) {
	_, diags := Parse("bad.af", `agent x { prompt: "oops }`)
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}
}

func TestParseRepeatStructure(t *testing.T) {
	src := `entry flow sql {
  repeat {
    generate as draft
    critic as verdict
  } until (verdict == pass, max 3)
}`
	astRoot, diags := Parse("repeat.af", src)
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	flow := astRoot.Decls[0].Flow
	step := flow.Items[0].Step.Repeat
	if step == nil {
		t.Fatal("expected Repeat")
	}
	if len(step.Body) != 2 {
		t.Fatalf("body len = %d", len(step.Body))
	}
	if step.Cond == nil || step.Cond.Enum != "pass" {
		t.Fatalf("cond = %+v", step.Cond)
	}
	if step.Max == nil || *step.Max != 3 {
		t.Fatalf("max = %v", step.Max)
	}
}

func TestParseReviewerAlias(t *testing.T) {
	src := `entry flow f {
  reviewer as review
}`
	astRoot, diags := Parse("alias.af", src)
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	atom := astRoot.Decls[0].Flow.Items[0].Step.Chain.Atoms[0]
	if atom.Alias == nil || *atom.Alias != "review" {
		t.Fatalf("alias = %v", atom.Alias)
	}
}

func TestParseReviewPosition(t *testing.T) {
	astRoot := parseFixture(t, "review.af")
	var buildAgent *ast.Agent
	for _, d := range astRoot.Decls {
		if d.Agent != nil && d.Agent.Name == "build" {
			buildAgent = d.Agent
			break
		}
	}
	if buildAgent == nil {
		t.Fatal("build agent not found")
	}
	if buildAgent.Pos.Line != 20 {
		t.Fatalf("build agent line = %d, want 20", buildAgent.Pos.Line)
	}
}

func TestParseLevelB(t *testing.T) {
	cases := []string{
		`flow F(a, b: Topic) { research }`,
		`entry flow f { summarize(topic) { research } }`,
		`entry flow f { parallel each items as x { research } }`,
		`entry flow f { branch it { case a -> done } }`,
		`use std.patterns as p`,
	}
	for _, src := range cases {
		_, diags := Parse("levelb.af", src)
		if diags.HasErrors() {
			t.Fatalf("parse %q: %v", src, diags)
		}
	}
}

func TestParseTruncatedReviewNoPanic(t *testing.T) {
	src := readFixture(t, "review.af")
	step := len(src) / 40
	if step < 1 {
		step = 1
	}
	for i := 1; i < len(src); i += step {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic at len %d: %v", i, r)
				}
			}()
			Parse("review.af", src[:i])
		}()
	}
}

func TestGoldenAST(t *testing.T) {
	fixtures := []string{
		"review.af",
		"pipeline.af",
		"research.af",
		"critic.af",
		"docs.af",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			astRoot := parseFixture(t, name)
			got, err := ast.MarshalSnapshot(astRoot)
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", strings.TrimSuffix(name, ".af")+".ast.json")
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden %s (run with -update): %v", golden, err)
			}
			if string(got) != string(want) {
				t.Fatalf("golden mismatch for %s", name)
			}
		})
	}
}

func TestParseReviewDiagnosticsClean(t *testing.T) {
	_, diags := Parse("examples/review.af", readFixture(t, "review.af"))
	if diags.HasErrors() {
		t.Fatal(diags)
	}
}

func TestParseSemicolonDiagnosticRender(t *testing.T) {
	_, diags := Parse("bad.af", "agent x { model: sonnet; }")
	if len(diags) == 0 {
		t.Fatal("expected diagnostics")
	}
	rendered := diag.Render("agent x { model: sonnet; }", diags)
	if !strings.Contains(rendered, "AF000") {
		t.Fatalf("render = %q", rendered)
	}
}

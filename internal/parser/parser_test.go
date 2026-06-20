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

var update = flag.Bool("update", false, "rewrite golden AST snapshots")

func TestParseSemicolon(t *testing.T) {
	t.Parallel()
	_, diags := Parse("test.af", "agent x { model: sonnet };")
	if !diags.HasErrors() {
		t.Fatal("expected parse error for semicolon")
	}
	if len(diags) != 1 || diags[0].Code != "AF000" {
		t.Fatalf("diags = %#v", diags)
	}
	if diags[0].Pos.Line <= 0 || diags[0].Pos.Column <= 0 {
		t.Fatalf("expected positioned diagnostic, got %#v", diags[0].Pos)
	}
}

func TestParseUnclosedBrace(t *testing.T) {
	t.Parallel()
	_, diags := Parse("test.af", "flow f { build")
	if !diags.HasErrors() || diags[0].Code != "AF000" {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestParseUnterminatedString(t *testing.T) {
	t.Parallel()
	_, diags := Parse("test.af", "agent x { prompt: \"hi")
	if !diags.HasErrors() || diags[0].Code != "AF000" {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestParseEmpty(t *testing.T) {
	t.Parallel()
	root, diags := Parse("empty.af", "")
	if diags.HasErrors() {
		t.Fatalf("unexpected diags: %#v", diags)
	}
	if root == nil {
		t.Fatal("expected non-nil AST")
	}
	if len(root.Decls) != 0 {
		t.Fatalf("expected zero decls, got %d", len(root.Decls))
	}
}

func TestParseTruncatedReviewAF(t *testing.T) {
	t.Parallel()
	src := readFixture(t, "review.af")
	for i := 1; i <= len(src); i++ {
		func() {
			defer func() {
				if recover() != nil {
					t.Fatalf("panic on prefix length %d", i)
				}
			}()
			Parse("review.af", src[:i])
		}()
	}
}

func TestParseRepeat(t *testing.T) {
	t.Parallel()
	src := `entry flow sql {
  repeat { generate as draft critic as verdict } until (verdict == pass, max 3)
}`
	root, diags := Parse("test.af", src)
	if diags.HasErrors() {
		t.Fatalf("diags = %#v", diags)
	}
	repeat := findRepeatStep(t, root)
	if repeat.Cond == nil || repeat.Cond.Enum != "pass" {
		t.Fatalf("cond = %#v", repeat.Cond)
	}
	if repeat.Max == nil || *repeat.Max != 3 {
		t.Fatalf("max = %#v", repeat.Max)
	}
	if len(repeat.Body) != 2 {
		t.Fatalf("expected 2 body steps, got %d", len(repeat.Body))
	}
	for i, want := range []string{"draft", "verdict"} {
		chain := repeat.Body[i].Chain
		if chain == nil || len(chain.Atoms) != 1 {
			t.Fatalf("step %d chain = %#v", i, chain)
		}
		if chain.Atoms[0].Alias == nil || *chain.Atoms[0].Alias != want {
			t.Fatalf("step %d alias = %#v, want %q", i, chain.Atoms[0].Alias, want)
		}
	}
}

func TestParseAtomAlias(t *testing.T) {
	t.Parallel()
	src := `flow f { reviewer as review }`
	root, diags := Parse("test.af", src)
	if diags.HasErrors() {
		t.Fatalf("diags = %#v", diags)
	}
	flow := root.Decls[0].Flow
	if flow == nil || len(flow.Items) != 1 {
		t.Fatal("expected one flow item")
	}
	chain := flow.Items[0].Step.Chain
	if chain == nil || len(chain.Atoms) != 1 {
		t.Fatal("expected single atom chain")
	}
	if chain.Atoms[0].Alias == nil || *chain.Atoms[0].Alias != "review" {
		t.Fatalf("alias = %#v", chain.Atoms[0].Alias)
	}
}

func TestParseReviewAFAgentPosition(t *testing.T) {
	t.Parallel()
	root, diags := Parse("review.af", readFixture(t, "review.af"))
	if diags.HasErrors() {
		t.Fatalf("diags = %#v", diags)
	}
	for _, decl := range root.Decls {
		if decl.Agent != nil && decl.Agent.Name == "build" {
			if decl.Agent.Pos.Line != 20 {
				t.Fatalf("agent build line = %d, want 20", decl.Agent.Pos.Line)
			}
			return
		}
	}
	t.Fatal("agent build not found")
}

func TestParseLevelBFlowParams(t *testing.T) {
	t.Parallel()
	assertParseOK(t, `flow F(a, b: Topic) { research }`, func(root *ast.AST) {
		if root.Decls[0].Flow.Params == nil || len(root.Decls[0].Flow.Params.Params) != 2 {
			t.Fatalf("params = %#v", root.Decls[0].Flow.Params)
		}
	})
}

func TestParseLevelBCallBlock(t *testing.T) {
	t.Parallel()
	assertParseOK(t, `flow f { summarize(topic) { research } }`, func(root *ast.AST) {
		atom := root.Decls[0].Flow.Items[0].Step.Chain.Atoms[0]
		if atom.Args == nil || atom.Block == nil {
			t.Fatalf("atom = %#v", atom)
		}
	})
}

func TestParseLevelBParallelEach(t *testing.T) {
	t.Parallel()
	assertParseOK(t, `flow f { parallel each items as x { research } }`, func(root *ast.AST) {
		par := root.Decls[0].Flow.Items[0].Step.Parallel
		if par.Each == nil || par.Each.As != "x" {
			t.Fatalf("each = %#v", par.Each)
		}
	})
}

func TestParseLevelBBranchIt(t *testing.T) {
	t.Parallel()
	assertParseOK(t, `flow f { branch it { case a -> done } }`, func(root *ast.AST) {
		vr := root.Decls[0].Flow.Items[0].Step.Branch.Value
		if !vr.It {
			t.Fatalf("value ref = %#v", vr)
		}
	})
}

func TestParseLevelBUseAlias(t *testing.T) {
	t.Parallel()
	assertParseOK(t, `use std.patterns as p`, func(root *ast.AST) {
		use := root.Decls[0].Use
		if use.Alias == nil || *use.Alias != "p" {
			t.Fatalf("alias = %#v", use.Alias)
		}
	})
}

func TestParseReviewAFSmoke(t *testing.T) {
	t.Parallel()
	root, diags := Parse("review.af", readFixture(t, "review.af"))
	if diags.HasErrors() {
		t.Fatalf("diags = %#v", diags)
	}
	if len(root.Decls) == 0 {
		t.Fatal("expected declarations")
	}
}

func TestGoldenAST(t *testing.T) {
	fixtures := []string{"review", "pipeline", "research", "critic", "docs"}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			src := readFixture(t, name+".af")
			root, diags := Parse(name+".af", src)
			if diags.HasErrors() {
				t.Fatalf("diags = %#v", diags)
			}
			got, err := ast.MarshalSnapshot(root)
			if err != nil {
				t.Fatal(err)
			}
			goldenPath := filepath.Join("testdata", name+".ast.json")
			if *update {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden: %v (run with -update)", err)
			}
			if string(got) != string(want) {
				t.Fatalf("snapshot mismatch for %s\n%s", name, diffLines(string(want), string(got)))
			}
		})
	}
}

func assertParseOK(t *testing.T, src string, check func(*ast.AST)) {
	t.Helper()
	root, diags := Parse("test.af", src)
	if diags.HasErrors() {
		t.Fatalf("diags = %#v", diags)
	}
	check(root)
}

func findRepeatStep(t *testing.T, root *ast.AST) *ast.Repeat {
	t.Helper()
	for _, decl := range root.Decls {
		if decl.Flow == nil {
			continue
		}
		for _, item := range decl.Flow.Items {
			if item.Step != nil && item.Step.Repeat != nil {
				return item.Step.Repeat
			}
		}
	}
	t.Fatal("repeat step not found")
	return nil
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "examples", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func diffLines(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	limit := len(wantLines)
	if len(gotLines) > limit {
		limit = len(gotLines)
	}
	var b strings.Builder
	for i := 0; i < limit; i++ {
		w := ""
		g := ""
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			b.WriteString("- " + w + "\n+ " + g + "\n")
		}
	}
	return b.String()
}

var _ diag.Diagnostics

package dot_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/dot"
	"github.com/Abhinand20/agentFlow/internal/ir"
	"github.com/Abhinand20/agentFlow/internal/pipeline"
)

func compileReview(t *testing.T) pipeline.Result {
	t.Helper()
	res, diags := pipeline.Compile(filepath.Join("..", "..", "examples", "review.af"))
	if diags.HasErrors() {
		t.Fatalf("review.af failed to compile: %#v", diags)
	}
	return res
}

func TestEmitIsValidDigraph(t *testing.T) {
	t.Parallel()
	out := dot.Emit(compileReview(t).IR)
	if !strings.HasPrefix(out, "digraph ") {
		t.Fatalf("DOT should start with digraph header, got:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "}") {
		t.Fatalf("DOT should close with brace, got:\n%s", out)
	}
	if strings.Count(out, "{") != strings.Count(out, "}") {
		t.Fatalf("unbalanced braces in DOT:\n%s", out)
	}
}

func TestEmitIncludesPrefixedSubflowLabels(t *testing.T) {
	t.Parallel()
	out := dot.Emit(compileReview(t).IR)
	for _, label := range []string{
		"code_review.build",
		"code_review.quality",
		"code_review.reviewer",
		"code_review.loop.build",
	} {
		if !strings.Contains(out, label) {
			t.Fatalf("DOT missing prefixed subflow label %q:\n%s", label, out)
		}
	}
}

func TestEmitHasGatherEdges(t *testing.T) {
	t.Parallel()
	out := dot.Emit(compileReview(t).IR)
	if !strings.Contains(out, `[label="gather"]`) {
		t.Fatalf("DOT missing gather edges:\n%s", out)
	}
	for _, branch := range []string{"lint", "security", "style"} {
		want := `"` + branch + `" -> "code_review.reviewer" [label="gather"];`
		if !strings.Contains(out, want) {
			t.Fatalf("DOT missing gather edge %q:\n%s", want, out)
		}
	}
}

func TestEmitHasSequentialAndBranchEdges(t *testing.T) {
	t.Parallel()
	out := dot.Emit(compileReview(t).IR)
	if !strings.Contains(out, `"code_review.build" -> "code_review.quality";`) {
		t.Fatalf("DOT missing sequential edge build->quality:\n%s", out)
	}
	if !strings.Contains(out, `-> "deploy" [label="approve"];`) {
		t.Fatalf("DOT missing approve branch edge to deploy:\n%s", out)
	}
}

func TestEmitGateNodeShape(t *testing.T) {
	t.Parallel()
	out := dot.Emit(compileReview(t).IR)
	if !strings.Contains(out, `"code_review.quality" [label="code_review.quality", shape=box]`) {
		t.Fatalf("gate node should render as a box:\n%s", out)
	}
}

func TestEmitDeterministic(t *testing.T) {
	t.Parallel()
	ir := compileReview(t).IR
	if dot.Emit(ir) != dot.Emit(ir) {
		t.Fatal("Emit must be deterministic")
	}
}

func TestEmitUnknownNodeKind(t *testing.T) {
	t.Parallel()
	p := compileReview(t).IR
	p.Flow.Root = ir.Node{Kind: ir.NodeKind("future")}
	out := dot.Emit(p)
	if !strings.Contains(out, `"unknown:future"`) {
		t.Fatalf("DOT should surface unsupported node kinds:\n%s", out)
	}
}

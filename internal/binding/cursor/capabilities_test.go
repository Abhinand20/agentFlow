package cursor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/binding/cursor"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/ir"
)

func loadReviewIR(t *testing.T) ir.Program {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "ir", "testdata", "review.ir.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ir.UnmarshalJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func codes(diags diag.Diagnostics) map[string]int {
	out := make(map[string]int)
	for _, d := range diags {
		out[d.Code]++
	}
	return out
}

func TestCapabilities(t *testing.T) {
	c := cursor.Binding()
	caps := c.Capabilities()

	wantTrue := []string{
		"command-trigger",
		"mcp",
	}
	for _, name := range wantTrue {
		if !caps[cursor.Capability(name)] {
			t.Fatalf("expected capability %q to be true", name)
		}
	}

	wantFalse := []string{
		"named-subagents",
		"hooks",
		"parallel-spawn",
		"blocking-gate",
		"output-parse",
		"loop-enforcement",
	}
	for _, name := range wantFalse {
		if caps[cursor.Capability(name)] {
			t.Fatalf("expected capability %q to be false", name)
		}
	}
}

func TestNegotiateReview(t *testing.T) {
	p := loadReviewIR(t)
	b := cursor.Binding()
	diags := cursor.Negotiate(p, b.Capabilities())
	got := codes(diags)

	for _, code := range []string{"AF300", "AF301", "AF302", "AF303"} {
		if got[code] == 0 {
			t.Fatalf("expected %s in review.af negotiation, got %v", code, got)
		}
	}
	if got["AF304"] != 0 {
		t.Fatalf("AF304 is a static binding caveat, not per-program: %v", got)
	}

	for _, d := range diags {
		if d.Severity != diag.Warning {
			t.Fatalf("expected warning, got %v for %s", d.Severity, d.Code)
		}
	}
}

func TestNegotiateMinimalNoParallel(t *testing.T) {
	p := ir.Program{
		Agents: []ir.Agent{{Name: "build", Prompt: "Build."}},
		Flow: ir.Flow{
			Root: ir.Node{
				Kind: ir.NodeSeq,
				Steps: []ir.Node{
					{
						Kind: ir.NodeStep,
						Step: &ir.StepRef{Decl: "build", Kind: "agent"},
					},
				},
			},
		},
	}
	diags := cursor.Negotiate(p, cursor.Binding().Capabilities())
	got := codes(diags)
	if got["AF300"] != 0 {
		t.Fatalf("unexpected AF300: %v", got)
	}
	if got["AF304"] != 0 {
		t.Fatalf("unexpected AF304: %v", got)
	}
}

func TestNegotiateBlockingGateOnly(t *testing.T) {
	p := ir.Program{
		Gates: []ir.Gate{
			{Name: "quality", Run: "scripts/test.sh", Behavior: "blocking"},
		},
		Flow: ir.Flow{
			Root: ir.Node{
				Kind: ir.NodeSeq,
				Steps: []ir.Node{
					{
						Kind: ir.NodeStep,
						Step: &ir.StepRef{Decl: "quality", Kind: "gate"},
					},
				},
			},
		},
	}
	diags := cursor.Negotiate(p, cursor.Binding().Capabilities())
	got := codes(diags)
	if got["AF303"] != 1 {
		t.Fatalf("expected one AF303, got %v", got)
	}
}

func TestNegotiateDerivedFromCapabilities(t *testing.T) {
	p := ir.Program{
		Flow: ir.Flow{
			Root: ir.Node{Kind: ir.NodeParallel, Branches: []ir.Node{
				{Kind: ir.NodeStep, Step: &ir.StepRef{Decl: "lint", Kind: "agent"}},
			}},
		},
	}
	allCaps := cursor.Binding().Capabilities()
	if len(cursor.Negotiate(p, allCaps)) == 0 {
		t.Fatal("expected AF300 when parallel-spawn unsupported")
	}

	supported := map[cursor.Capability]bool{}
	for k, v := range allCaps {
		supported[k] = v
	}
	supported[cursor.CapParallelSpawn] = true
	if len(cursor.Negotiate(p, supported)) != 0 {
		t.Fatal("expected no AF300 when parallel-spawn capability enabled")
	}
}

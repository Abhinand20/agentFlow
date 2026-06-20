package sema

import (
	"testing"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
	"github.com/Abhinand20/agentFlow/internal/parser"
)

func resolveSrc(t *testing.T, src string) (*model.Program, diag.Diagnostics) {
	t.Helper()
	root, diags := parser.Parse("test.af", src)
	if diags.HasErrors() {
		t.Fatalf("parse failed: %#v", diags)
	}
	prog, rdiags := Resolve(root, ".")
	return prog, rdiags
}

func hasCode(diags diag.Diagnostics, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestDeclarePassRecordsAllNamesInOrder(t *testing.T) {
	src := `use anthropic { kind: model-provider models: [opus] }
type Verdict = approve | reject
agent a { model: opus }
gate g { run: "x.sh" }
entry flow f { a }`
	prog, _ := resolveSrc(t, src)
	if len(prog.Order) != 5 {
		t.Fatalf("Order len = %d, want 5", len(prog.Order))
	}
	if prog.Capabilities["anthropic"] == nil || prog.Types["Verdict"] == nil ||
		prog.Agents["a"] == nil || prog.Gates["g"] == nil || prog.Flows["f"] == nil {
		t.Fatalf("missing symbol-table entries: %#v", prog)
	}
}

func TestDuplicateNameLastWinsBothInOrder(t *testing.T) {
	src := `agent a { model: opus }
agent a { model: sonnet }`
	prog, _ := resolveSrc(t, src)
	if prog.Agents["a"].ModelAlias != "sonnet" {
		t.Fatalf("last decl should win in map: %#v", prog.Agents["a"])
	}
	n := 0
	for _, r := range prog.Order {
		if r.Name == "a" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("Order should record both duplicates, got %d", n)
	}
}

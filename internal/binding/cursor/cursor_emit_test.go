package cursor_test

import (
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/binding/cursor"
)

func TestRegistration(t *testing.T) {
	got, ok := binding.Get("cursor")
	if !ok {
		t.Fatal("expected cursor binding to be registered")
	}
	if got.Name() != "cursor" {
		t.Fatalf("Name() = %q, want cursor", got.Name())
	}
	if cursor.Binding().Name() != "cursor" {
		t.Fatalf("Binding() Name mismatch")
	}
}

func TestEmitReviewPaths(t *testing.T) {
	p := loadReviewIR(t)
	b := cursor.Binding()
	fs, diags := b.Emit(p)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	wantPaths := []string{
		".cursor/commands/ship.md",
		".cursor/mcp.json",
		".cursor/rules/build.mdc",
		".cursor/rules/deploy.mdc",
		".cursor/rules/lint.mdc",
		".cursor/rules/notify_author.mdc",
		".cursor/rules/reviewer.mdc",
		".cursor/rules/security.mdc",
		".cursor/rules/style.mdc",
	}
	gotPaths := fs.Paths()
	if len(gotPaths) != len(wantPaths) {
		t.Fatalf("path count = %d, want %d\n got: %v", len(gotPaths), len(wantPaths), gotPaths)
	}
	for i, want := range wantPaths {
		if gotPaths[i] != want {
			t.Fatalf("path[%d] = %q, want %q", i, gotPaths[i], want)
		}
	}
}

func TestEmitReviewContents(t *testing.T) {
	p := loadReviewIR(t)
	fs, _ := cursor.Binding().Emit(p)

	ship, ok := fs.Get(".cursor/commands/ship.md")
	if !ok {
		t.Fatal("missing ship.md")
	}
	shipText := string(ship)
	for _, want := range []string{
		"code_review",
		"revise",
		"step `build`",
		"one after another",
	} {
		if !strings.Contains(shipText, want) {
			t.Fatalf("ship.md missing %q", want)
		}
	}

	reviewer, ok := fs.Get(".cursor/rules/reviewer.mdc")
	if !ok {
		t.Fatal("missing reviewer.mdc")
	}
	reviewerText := string(reviewer)
	for _, want := range []string{
		"```agentflow-output",
		"out: <value>",
		"approve, revise, reject",
		"alwaysApply: false",
	} {
		if !strings.Contains(reviewerText, want) {
			t.Fatalf("reviewer.mdc missing %q", want)
		}
	}

	mcp, ok := fs.Get(".cursor/mcp.json")
	if !ok {
		t.Fatal("missing mcp.json")
	}
	mcpText := string(mcp)
	for _, want := range []string{
		"github",
		"npx",
		"@modelcontextprotocol/server-github",
	} {
		if !strings.Contains(mcpText, want) {
			t.Fatalf("mcp.json missing %q", want)
		}
	}
}

func TestEmitReviewNegotiation(t *testing.T) {
	p := loadReviewIR(t)
	_, diags := cursor.Binding().Emit(p)
	got := codes(diags)
	for _, code := range []string{"AF300", "AF301", "AF302", "AF303", "AF304"} {
		if got[code] == 0 {
			t.Fatalf("expected %s during Emit", code)
		}
	}
}

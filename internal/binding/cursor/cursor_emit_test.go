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
		"<!-- agentflow: trigger=/ship flow=ship in=Ticket -->",
		"code_review",
		"revise",
		"step `build`",
		"one after another",
	} {
		if !strings.Contains(shipText, want) {
			t.Fatalf("ship.md missing %q", want)
		}
	}

	build, ok := fs.Get(".cursor/rules/build.mdc")
	if !ok {
		t.Fatal("missing build.mdc")
	}
	buildText := string(build)
	for _, want := range []string{
		"<!-- agentflow: model=sonnet tools=github:get_pr -->",
		"alwaysApply: false",
	} {
		if !strings.Contains(buildText, want) {
			t.Fatalf("build.mdc missing %q", want)
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
		"<!-- agentflow: model=opus -->",
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
		"stdio",
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
	for _, code := range []string{"AF300", "AF301", "AF302", "AF303", "AF305", "AF306"} {
		if got[code] == 0 {
			t.Fatalf("expected %s during Emit", code)
		}
	}
	if got["AF304"] != 0 {
		t.Fatalf("AF304 is static binding caveat, not per Emit: %v", got)
	}
}

func TestEmitReviewFlowInputVocabulary(t *testing.T) {
	v := cursor.Vocabulary()
	got := v.Arg("Ticket")
	if got != "$1" {
		t.Fatalf("Arg(Ticket) = %q, want $1", got)
	}
	// review.af has no agent with in: Ticket; command metadata carries in=Ticket instead.
	p := loadReviewIR(t)
	fs, _ := cursor.Binding().Emit(p)
	content, ok := fs.Get(".cursor/commands/ship.md")
	if !ok {
		t.Fatal("missing ship.md")
	}
	if !strings.Contains(string(content), "in=Ticket") {
		t.Fatal("expected entry InType in command metadata")
	}
}

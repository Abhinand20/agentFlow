package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/binding/cursor"
	"github.com/Abhinand20/agentFlow/internal/pipeline"
)

// TestE2E_ReviewAf_CursorTree drives compile -> cursor.Emit -> Flush in-process
// and checks integration-specific content (gate script, output protocol, MCP).
// Byte-identical FS goldens live in internal/binding/cursor/golden_test.go.
func TestE2E_ReviewAf_CursorTree(t *testing.T) {
	t.Parallel()

	res, diags := pipeline.Compile(reviewAf)
	if diags.HasErrors() {
		t.Fatalf("review.af must compile clean: %#v", diags)
	}

	hostFS, emitDiags := cursor.Binding().Emit(res.IR)
	if emitDiags.HasErrors() {
		t.Fatalf("cursor emit produced errors: %#v", emitDiags)
	}
	out := t.TempDir()
	if err := hostFS.Flush(out); err != nil {
		t.Fatalf("flush: %v", err)
	}

	wantPaths := []string{
		".cursor/commands/ship.md",
		".cursor/agents/build.md",
		".cursor/agents/deploy.md",
		".cursor/agents/lint.md",
		".cursor/agents/notify_author.md",
		".cursor/agents/reviewer.md",
		".cursor/agents/security.md",
		".cursor/agents/style.md",
		".cursor/mcp.json",
	}
	for _, rel := range wantPaths {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing emitted path %s: %v", rel, err)
		}
	}

	ship := readFile(t, out, ".cursor/commands/ship.md")
	if !strings.Contains(ship, "scripts/test.sh") {
		t.Errorf("ship.md should reference the gate script:\n%s", ship)
	}
	if !strings.Contains(ship, "go back to step `build`") {
		t.Errorf("ship.md gate should retry to control label build:\n%s", ship)
	}
	if !strings.Contains(ship, "agentflow-output") {
		t.Errorf("ship.md should instruct reading the agentflow-output block:\n%s", ship)
	}

	reviewer := readFile(t, out, ".cursor/agents/reviewer.md")
	if !strings.Contains(reviewer, "```agentflow-output") {
		t.Errorf("reviewer.md should embed the agentflow-output protocol block:\n%s", reviewer)
	}
	if !strings.Contains(reviewer, "out: <value>") {
		t.Errorf("reviewer.md output block should carry the out: <value> contract:\n%s", reviewer)
	}

	mcp := readFile(t, out, ".cursor/mcp.json")
	if !strings.Contains(mcp, "github") {
		t.Errorf(".cursor/mcp.json should declare the github server:\n%s", mcp)
	}
	if !strings.Contains(mcp, "@modelcontextprotocol/server-github") {
		t.Errorf(".cursor/mcp.json should carry the github server args:\n%s", mcp)
	}
}

func readFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/binding/cursor"
	"github.com/Abhinand20/agentFlow/internal/pipeline"
)

// afBin is the compiled `af` binary used by the real-binary E2E test. Building
// once (rather than `go run` per call) keeps the binary's real exit codes —
// `go run` collapses any non-zero program exit to status 1 — and is faster.
var afBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "af-e2e-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	afBin = filepath.Join(dir, "af")
	if out, err := exec.Command("go", "build", "-o", afBin, ".").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build af: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// cursorGoldenDir is the single source of truth for the Cursor host tree
// emitted from review.af (owned by the cursor binding package).
const cursorGoldenDir = "../../internal/binding/cursor/testdata"

// TestE2E_ReviewAf_CursorTree drives the whole pipeline in-process
// (compile -> cursor.Emit -> Flush) and asserts the final host tree on disk:
// the expected paths exist, key content goldens hold, and every emitted file
// is byte-identical to the canonical cursor binding golden.
func TestE2E_ReviewAf_CursorTree(t *testing.T) {
	t.Parallel()

	res, diags := pipeline.Compile(reviewAf)
	if diags.HasErrors() {
		t.Fatalf("review.af must compile clean: %#v", diags)
	}

	hostFS, _ := cursor.Binding().Emit(res.IR)
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

	for _, rel := range hostFS.Paths() {
		got := readFile(t, out, rel)
		want, err := os.ReadFile(filepath.Join(cursorGoldenDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read golden %s: %v", rel, err)
		}
		if got != string(want) {
			t.Errorf("E2E tree drifted from cursor golden for %s", rel)
		}
	}
}

// TestE2E_RealBinary exercises the full CLI contract through the compiled
// binary so flag parsing, binding registration, and exit codes are validated
// end to end.
func TestE2E_RealBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-binary E2E in -short mode")
	}
	t.Parallel()

	out := t.TempDir()

	stdout, stderr, code := goRun(t, "build", reviewAf, "--target", "cursor", "--out", out)
	if code != 0 {
		t.Fatalf("build exit = %d, stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, filepath.Join(out, ".cursor", "commands", "ship.md")) {
		t.Fatalf("build summary should list written paths:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(out, ".cursor", "mcp.json")); err != nil {
		t.Fatalf("real binary did not write .cursor/mcp.json: %v", err)
	}
	if !strings.Contains(stderr, "AF3") {
		t.Fatalf("expected AF3xx warnings from cursor binding on stderr:\n%s", stderr)
	}

	vout, _, vcode := goRun(t, "validate", reviewAf)
	if vcode != 0 {
		t.Fatalf("validate exit = %d, want 0", vcode)
	}
	if !strings.Contains(vout, "ok") {
		t.Fatalf("validate should print ok:\n%s", vout)
	}

	gout, _, gcode := goRun(t, "graph", reviewAf)
	if gcode != 0 {
		t.Fatalf("graph exit = %d, want 0", gcode)
	}
	if !strings.Contains(gout, "code_review.build") || !strings.Contains(gout, `[label="gather"]`) {
		t.Fatalf("graph DOT missing expected content:\n%s", gout)
	}

	_, berr, bcode := goRun(t, "build", reviewAf, "--target", "nope")
	if bcode != 2 {
		t.Fatalf("unknown target exit = %d, want 2", bcode)
	}
	if !strings.Contains(berr, "cursor") {
		t.Fatalf("unknown-target error should list available targets:\n%s", berr)
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

func goRun(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(afBin, args...)
	var o, e bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &e
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("run af %v: %v\nstderr:\n%s", args, err, e.String())
		}
		code = ee.ExitCode()
	}
	return o.String(), e.String(), code
}

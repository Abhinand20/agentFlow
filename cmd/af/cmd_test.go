package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphReviewAf(t *testing.T) {
	t.Parallel()
	out, _, code := runInProc("graph", reviewAf)
	if code != 0 {
		t.Fatalf("graph exit = %d, want 0", code)
	}
	if !strings.Contains(out, "digraph ") {
		t.Fatalf("graph should emit DOT:\n%s", out)
	}
	if !strings.Contains(out, "code_review.build") {
		t.Fatalf("graph DOT missing code_review.build:\n%s", out)
	}
	if !strings.Contains(out, `[label="gather"]`) {
		t.Fatalf("graph DOT missing gather edges:\n%s", out)
	}
}

func TestGraphMissingFileExit1(t *testing.T) {
	t.Parallel()
	_, _, code := runInProc("graph", filepath.Join("..", "..", "examples", "does-not-exist.af"))
	if code != 1 {
		t.Fatalf("graph missing file exit = %d, want 1", code)
	}
}

func TestGraphUsageExit2(t *testing.T) {
	t.Parallel()
	_, _, code := runInProc("graph")
	if code != 2 {
		t.Fatalf("graph without file exit = %d, want 2", code)
	}
}

func TestBuildCursorTarget(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	stdout, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir)
	if code != 0 {
		t.Fatalf("build exit = %d, want 0\nstderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, filepath.Join(outDir, ".cursor", "commands", "ship.md")) {
		t.Fatalf("build summary should list written paths:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(outDir, ".cursor", "mcp.json")); err != nil {
		t.Fatalf("build did not write .cursor/mcp.json: %v", err)
	}
	if !strings.Contains(stderr, "AF3") {
		t.Fatalf("expected AF3xx warnings from cursor binding on stderr:\n%s", stderr)
	}
}

func TestBuildEmitIR(t *testing.T) {
	t.Parallel()
	out, _, code := runInProc("build", reviewAf, "--emit-ir")
	if code != 0 {
		t.Fatalf("build --emit-ir exit = %d, want 0", code)
	}
	if !strings.Contains(out, `"flowName"`) && !strings.Contains(out, `"FlowName"`) {
		t.Fatalf("emit-ir should print IR JSON:\n%s", out)
	}
}

func TestBuildMissingTargetExit2(t *testing.T) {
	t.Parallel()
	_, errOut, code := runInProc("build", reviewAf)
	if code != 2 {
		t.Fatalf("build without --target exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "--target is required") {
		t.Fatalf("expected --target required error:\n%s", errOut)
	}
}

func TestBuildUnknownTargetExit2(t *testing.T) {
	t.Parallel()
	_, errOut, code := runInProc("build", reviewAf, "--target", "nope")
	if code != 2 {
		t.Fatalf("unknown target exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "cursor") {
		t.Fatalf("unknown-target error should list available targets:\n%s", errOut)
	}
}

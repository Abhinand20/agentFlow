package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

const reviewAf = "../../examples/review.af"
const pipelineAf = "../../examples/pipeline.af"

// runInProc invokes the dispatcher in-process and returns stdout, stderr, and
// the exit code so tests can assert the exit policy precisely.
func runInProc(args ...string) (stdout, stderr string, code int) {
	var out, err bytes.Buffer
	code = run(args, &out, &err)
	return out.String(), err.String(), code
}

func TestUsageNoArgs(t *testing.T) {
	t.Parallel()
	out, _, code := runInProc()
	if code != 0 {
		t.Fatalf("no-args exit = %d, want 0", code)
	}
	if !strings.Contains(out, "AgentFlow compiler") {
		t.Fatalf("usage missing banner: %s", out)
	}
	for _, cmd := range []string{"validate", "build", "clean", "versions", "graph"} {
		if !strings.Contains(out, cmd) {
			t.Fatalf("usage missing %q command: %s", cmd, out)
		}
	}
}

func TestHelpFlag(t *testing.T) {
	t.Parallel()
	out, _, code := runInProc("--help")
	if code != 0 {
		t.Fatalf("--help exit = %d, want 0", code)
	}
	if !strings.Contains(out, "AgentFlow compiler") {
		t.Fatalf("--help missing banner: %s", out)
	}
}

func TestUnknownCommandExit2(t *testing.T) {
	t.Parallel()
	_, errOut, code := runInProc("nope")
	if code != 2 {
		t.Fatalf("unknown command exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "unknown command") {
		t.Fatalf("stderr = %q", errOut)
	}
}

func TestValidateReviewAf(t *testing.T) {
	t.Parallel()
	out, _, code := runInProc("validate", reviewAf)
	if code != 0 {
		t.Fatalf("validate exit = %d, want 0", code)
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("validate should print ok:\n%s", out)
	}
}

func TestValidateMissingFileExit1(t *testing.T) {
	t.Parallel()
	_, errOut, code := runInProc("validate", filepath.Join("..", "..", "examples", "does-not-exist.af"))
	if code != 1 {
		t.Fatalf("validate missing file exit = %d, want 1", code)
	}
	if !strings.Contains(errOut, "AF000") {
		t.Fatalf("expected AF000 read error on stderr:\n%s", errOut)
	}
}

func TestValidateUsageExit2(t *testing.T) {
	t.Parallel()
	_, _, code := runInProc("validate")
	if code != 2 {
		t.Fatalf("validate without file exit = %d, want 2", code)
	}
}

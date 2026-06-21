//go:build integration

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
)

// afBin is the compiled `af` binary used by integration tests. Building once
// (rather than `go run` per call) preserves real exit codes and is faster.
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

// TestE2E_RealBinary exercises the full CLI contract through the compiled
// binary. Build output/AF3xx contract is covered in cmd_test.go (in-process).
func TestE2E_RealBinary(t *testing.T) {
	t.Parallel()

	vout, _, vcode := runAFBin(t, "validate", reviewAf)
	if vcode != 0 {
		t.Fatalf("validate exit = %d, want 0", vcode)
	}
	if !strings.Contains(vout, "ok") {
		t.Fatalf("validate should print ok:\n%s", vout)
	}

	gout, _, gcode := runAFBin(t, "graph", reviewAf)
	if gcode != 0 {
		t.Fatalf("graph exit = %d, want 0", gcode)
	}
	if !strings.Contains(gout, "code_review.build") || !strings.Contains(gout, `[label="gather"]`) {
		t.Fatalf("graph DOT missing expected content:\n%s", gout)
	}

	_, berr, bcode := runAFBin(t, "build", reviewAf, "--target", "nope")
	if bcode != 2 {
		t.Fatalf("unknown target exit = %d, want 2", bcode)
	}
	if !strings.Contains(berr, "cursor") {
		t.Fatalf("unknown-target error should list available targets:\n%s", berr)
	}
}

func runAFBin(t *testing.T, args ...string) (stdout, stderr string, code int) {
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

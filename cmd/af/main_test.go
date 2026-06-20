package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMainUsage(t *testing.T) {
	t.Parallel()
	out := runAF(t)
	if !bytes.Contains(out, []byte("AgentFlow compiler")) {
		t.Fatalf("usage missing banner: %s", out)
	}
	if !bytes.Contains(out, []byte("validate")) {
		t.Fatalf("usage missing validate: %s", out)
	}
}

func TestValidateNotImplemented(t *testing.T) {
	t.Parallel()
	out := runAF(t, "validate")
	if !bytes.Contains(out, []byte("validate: not implemented")) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestBuildNotImplemented(t *testing.T) {
	t.Parallel()
	out := runAF(t, "build")
	if !bytes.Contains(out, []byte("build: not implemented")) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestGraphNotImplemented(t *testing.T) {
	t.Parallel()
	out := runAF(t, "graph")
	if !bytes.Contains(out, []byte("graph: not implemented")) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "run", ".", "nope")
	cmd.Dir = afDir(t)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero exit for unknown command")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("unknown command")) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func runAF(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Dir = afDir(t)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run: %v\noutput: %s", err, stdout.Bytes())
	}
	return stdout.Bytes()
}

func afDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "..", "..", "cmd", "af")
}

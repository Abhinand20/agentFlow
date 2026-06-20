package sema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
	"github.com/Abhinand20/agentFlow/internal/parser"
)

func resolveDir(t *testing.T, dir, src string) (*model.Program, diag.Diagnostics) {
	t.Helper()
	root, d := parser.Parse("test.af", src)
	if d.HasErrors() {
		t.Fatalf("parse: %#v", d)
	}
	return Resolve(root, dir)
}

func TestPromptInlineVerbatim(t *testing.T) {
	prog, _ := resolveSrc(t, `agent a { model: opus prompt: "Do the thing." }`)
	ag := prog.Agents["a"]
	if ag.Prompt != "Do the thing." || ag.PromptFromFile {
		t.Fatalf("agent = %#v", ag)
	}
}

func TestPromptFromMdPathRead(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "p.md"), []byte("FILE BODY"), 0o644)
	prog, _ := resolveDir(t, dir, `agent a { model: opus prompt: "p.md" }`)
	ag := prog.Agents["a"]
	if ag.Prompt != "FILE BODY" || !ag.PromptFromFile || ag.PromptPath != "p.md" || !ag.Resolution.OK {
		t.Fatalf("agent = %#v", ag)
	}
}

func TestPromptAbsolutePathReason(t *testing.T) {
	prog, _ := resolveSrc(t, `agent a { model: opus prompt-file: "/etc/passwd" }`)
	if prog.Agents["a"].Resolution.Reason != "absolute" {
		t.Fatalf("reason = %q", prog.Agents["a"].Resolution.Reason)
	}
}

func TestPromptEscapesTreeReason(t *testing.T) {
	dir := t.TempDir()
	prog, _ := resolveDir(t, dir, `agent a { model: opus prompt-file: "../secrets.md" }`)
	if prog.Agents["a"].Resolution.Reason != "escapes-tree" {
		t.Fatalf("reason = %q", prog.Agents["a"].Resolution.Reason)
	}
}

func TestPromptMissingReason(t *testing.T) {
	dir := t.TempDir()
	prog, _ := resolveDir(t, dir, `agent a { model: opus prompt-file: "nope.md" }`)
	if prog.Agents["a"].Resolution.Reason != "missing" {
		t.Fatalf("reason = %q", prog.Agents["a"].Resolution.Reason)
	}
}

func TestPromptConflictReason(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "p.md"), []byte("BODY"), 0o644)
	prog, _ := resolveDir(t, dir, `agent a { model: opus prompt: "inline" prompt-file: "p.md" }`)
	ag := prog.Agents["a"]
	if ag.Resolution.Reason != "conflict" || ag.Prompt != "BODY" {
		t.Fatalf("agent = %#v", ag)
	}
}

func TestPromptPreservesDollarName(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "p.md"), []byte("Hello ${NAME}"), 0o644)
	prog, _ := resolveDir(t, dir, `agent a { model: opus prompt-file: "p.md" }`)
	if prog.Agents["a"].Prompt != "Hello ${NAME}" {
		t.Fatalf("prompt = %q", prog.Agents["a"].Prompt)
	}
}

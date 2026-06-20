package emit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/emit"
)

func TestFSWriteGetPathsSorted(t *testing.T) {
	t.Parallel()

	fs := emit.NewFS()
	fs.Write("b.txt", []byte("b"))
	fs.Write("a.txt", []byte("a"))
	fs.Write("c/nested.txt", []byte("nested"))

	got := fs.Paths()
	want := []string{"a.txt", "b.txt", "c/nested.txt"}
	if len(got) != len(want) {
		t.Fatalf("Paths() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Paths()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	content, ok := fs.Get("a.txt")
	if !ok || string(content) != "a" {
		t.Fatalf("Get(a.txt) = %q, %v", content, ok)
	}
}

func TestFSWriteReplacesContent(t *testing.T) {
	t.Parallel()

	fs := emit.NewFS()
	fs.Write("x.txt", []byte("v1"))
	fs.Write("x.txt", []byte("v2"))

	content, _ := fs.Get("x.txt")
	if string(content) != "v2" {
		t.Fatalf("got %q, want v2", content)
	}
	if len(fs.Paths()) != 1 {
		t.Fatalf("expected one path, got %d", len(fs.Paths()))
	}
}

func TestFSFlushRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	fs := emit.NewFS()
	fs.Write(".claude/agents/reviewer.md", []byte("# Reviewer\n"))
	fs.Write(".claude/commands/ship.md", []byte("# Ship\n"))

	if err := fs.Flush(dir); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	for _, rel := range []string{".claude/agents/reviewer.md", ".claude/commands/ship.md"} {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		got, err := os.ReadFile(full)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", rel, err)
		}
		want, _ := fs.Get(rel)
		if string(got) != string(want) {
			t.Fatalf("%s: got %q, want %q", rel, got, want)
		}
	}
}

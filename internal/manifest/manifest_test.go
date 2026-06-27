package manifest_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Abhinand20/agentFlow/internal/emit"
	"github.com/Abhinand20/agentFlow/internal/manifest"
)

func fixedNow() func() time.Time {
	return func() time.Time {
		return time.Date(2026, 6, 27, 6, 0, 0, 0, time.UTC)
	}
}

func TestSlugStable(t *testing.T) {
	t.Parallel()
	a := manifest.Slug("/tmp/examples/review.af")
	b := manifest.Slug("/tmp/examples/review.af")
	if a != b {
		t.Fatalf("slug not stable: %q vs %q", a, b)
	}
	if a == "review" {
		t.Fatalf("slug should include hash suffix, got %q", a)
	}
}

func TestBuildDeterministic(t *testing.T) {
	t.Parallel()
	fs := emit.NewFS()
	fs.Write(".cursor/agents/reviewer.md", []byte("agent"))
	fs.Write(".cursor/commands/ship.md", []byte("cmd"))

	m := manifest.Build(manifest.BuildOptions{
		Target:       "cursor",
		SourcePath:   "examples/review.af",
		SourceSHA256: "abc123",
		IRHash:       "irhash",
		FS:           fs,
		Now:          fixedNow(),
		ToolVersion:  "test",
	})

	if m.Version != 1 {
		t.Fatalf("version = %d, want 1", m.Version)
	}
	if m.GeneratedAt != "2026-06-27T06:00:00Z" {
		t.Fatalf("generatedAt = %q", m.GeneratedAt)
	}
	if len(m.Artifacts) != 2 {
		t.Fatalf("artifacts = %d, want 2", len(m.Artifacts))
	}
	if len(m.History) != 1 || m.History[0].Version != 1 {
		t.Fatalf("history head = %#v", m.History)
	}
}

func TestBuildVersionBumpAndHistoryCap(t *testing.T) {
	t.Parallel()
	fs := emit.NewFS()
	fs.Write(".cursor/agents/a.md", []byte("a"))

	prior := manifest.Build(manifest.BuildOptions{
		Target:       "cursor",
		SourcePath:   "examples/review.af",
		SourceSHA256: "v1",
		IRHash:       "ir1",
		FS:           fs,
		Now:          fixedNow(),
		ToolVersion:  "test",
	})
	prior.History = make([]manifest.BuildRecord, manifest.HistoryCap)
	for i := range prior.History {
		prior.History[i].Version = i + 1
	}
	prior.Version = manifest.HistoryCap

	m := manifest.Build(manifest.BuildOptions{
		Target:       "cursor",
		SourcePath:   "examples/review.af",
		SourceSHA256: "v2",
		IRHash:       "ir2",
		FS:           fs,
		Prior:        &prior,
		Now:          fixedNow(),
		ToolVersion:  "test",
	})

	if m.Version != manifest.HistoryCap+1 {
		t.Fatalf("version = %d, want %d", m.Version, manifest.HistoryCap+1)
	}
	if len(m.History) != manifest.HistoryCap {
		t.Fatalf("history len = %d, want %d", len(m.History), manifest.HistoryCap)
	}
	if m.History[0].IRHash != "ir2" {
		t.Fatalf("history head irHash = %q", m.History[0].IRHash)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	t.Parallel()
	fs := emit.NewFS()
	fs.Write(".cursor/agents/a.md", []byte("a"))
	m := manifest.Build(manifest.BuildOptions{
		Target:       "cursor",
		SourcePath:   "examples/review.af",
		SourceSHA256: "abc",
		IRHash:       "ir",
		FS:           fs,
		Now:          fixedNow(),
		ToolVersion:  "test",
	})
	data, err := manifest.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := manifest.Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source.Path != m.Source.Path || got.Version != m.Version {
		t.Fatalf("round-trip mismatch: %#v", got)
	}
}

func TestDiff(t *testing.T) {
	t.Parallel()
	a := manifest.BuildRecord{
		Artifacts: []manifest.HistoryArtifact{
			{Path: ".cursor/agents/a.md", SHA256: "1"},
			{Path: ".cursor/agents/b.md", SHA256: "2"},
		},
	}
	b := manifest.BuildRecord{
		Artifacts: []manifest.HistoryArtifact{
			{Path: ".cursor/agents/a.md", SHA256: "1"},
			{Path: ".cursor/agents/c.md", SHA256: "3"},
		},
	}
	changes := manifest.Diff(a, b)
	if len(changes.Removed) != 1 || changes.Removed[0] != ".cursor/agents/b.md" {
		t.Fatalf("removed = %#v", changes.Removed)
	}
	if len(changes.Added) != 1 || changes.Added[0] != ".cursor/agents/c.md" {
		t.Fatalf("added = %#v", changes.Added)
	}
}

func TestDriftCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rel := ".cursor/agents/reviewer.md"
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("hello")
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatal(err)
	}

	m := &manifest.Manifest{
		Artifacts: []manifest.Artifact{
			{Path: rel, SHA256: manifest.HashBytes(content)},
			{Path: ".cursor/agents/missing.md", SHA256: "deadbeef"},
			{Path: ".cursor/agents/modified.md", SHA256: "aaa"},
		},
	}
	modFull := filepath.Join(dir, filepath.FromSlash(".cursor/agents/modified.md"))
	if err := os.WriteFile(modFull, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := manifest.DriftCheck(m, dir)
	if len(report.Clean) != 1 {
		t.Fatalf("clean = %#v", report.Clean)
	}
	if len(report.Missing) != 1 {
		t.Fatalf("missing = %#v", report.Missing)
	}
	if len(report.Modified) != 1 {
		t.Fatalf("modified = %#v", report.Modified)
	}
}

func TestLoadAllAndOverlaps(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeManifest(t, dir, "cursor", "examples/a.af", []string{".cursor/agents/shared.md"})
	writeManifest(t, dir, "cursor", "examples/b.af", []string{".cursor/agents/shared.md", ".cursor/agents/b-only.md"})

	all, err := manifest.LoadAll(dir, "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("load all = %d, want 2", len(all))
	}

	conflicts := manifest.Overlaps(all[1], all)
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %#v", conflicts)
	}
}

func writeManifest(t *testing.T, dir, target, source string, paths []string) {
	t.Helper()
	fs := emit.NewFS()
	for _, p := range paths {
		fs.Write(p, []byte("x"))
	}
	m := manifest.Build(manifest.BuildOptions{
		Target:       target,
		SourcePath:   source,
		SourceSHA256: manifest.HashBytes([]byte(source)),
		IRHash:       "ir",
		FS:           fs,
		Now:          fixedNow(),
		ToolVersion:  "test",
	})
	data, err := manifest.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	rel := manifest.ManifestRelPath(target, source)
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/emit"
	"github.com/Abhinand20/agentFlow/internal/manifest"
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

func TestGraphUnknownTargetExit2(t *testing.T) {
	t.Parallel()
	_, errOut, code := runInProc("graph", reviewAf, "--target", "nope")
	if code != 2 {
		t.Fatalf("graph unknown target exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "cursor") {
		t.Fatalf("graph unknown-target error should list available targets:\n%s", errOut)
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
	manifestPaths, err := filepath.Glob(filepath.Join(outDir, ".cursor", ".agentflow", "manifests", "*.json"))
	if err != nil || len(manifestPaths) != 1 {
		t.Fatalf("expected one manifest file, got %v err=%v", manifestPaths, err)
	}
	data, err := os.ReadFile(manifestPaths[0])
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("manifest should be valid JSON:\n%s", data)
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
	trimmed := strings.TrimSpace(out)
	if !json.Valid([]byte(trimmed)) {
		t.Fatalf("emit-ir should print valid JSON:\n%s", out)
	}
	if !strings.Contains(trimmed, `"FlowName": "ship"`) {
		t.Fatalf("emit-ir JSON should include entry FlowName:\n%s", out)
	}
	if !strings.Contains(trimmed, `"Trigger": "/ship"`) {
		t.Fatalf("emit-ir JSON should include entry Trigger:\n%s", out)
	}
}

func TestBuildEmitIRIgnoresTarget(t *testing.T) {
	t.Parallel()
	_, stderr, code := runInProc("build", reviewAf, "--emit-ir", "--target", "cursor")
	if code != 0 {
		t.Fatalf("build --emit-ir with --target exit = %d, want 0", code)
	}
	if !strings.Contains(stderr, "--emit-ir ignores --target") {
		t.Fatalf("expected --emit-ir ignores --target warning:\n%s", stderr)
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

func TestCleanRequiresSourceOrAll(t *testing.T) {
	t.Parallel()
	_, errOut, code := runInProc("clean", "--target", "cursor")
	if code != 2 {
		t.Fatalf("clean without source exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "specify an .af source or --all") {
		t.Fatalf("expected usage error:\n%s", errOut)
	}
}

func TestCleanSourceScopedIsolation(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("build review exit = %d\n%s", code, stderr)
	}
	if _, stderr, code := runInProc("build", pipelineAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("build pipeline exit = %d\n%s", code, stderr)
	}

	if _, err := os.Stat(filepath.Join(outDir, ".cursor", "commands", "write.md")); err != nil {
		t.Fatalf("pipeline command missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, ".cursor", "commands", "ship.md")); err != nil {
		t.Fatalf("review command missing: %v", err)
	}

	if _, stderr, code := runInProc("clean", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("clean review exit = %d\n%s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(outDir, ".cursor", "commands", "ship.md")); !os.IsNotExist(err) {
		t.Fatalf("review command should be removed")
	}
	if _, err := os.Stat(filepath.Join(outDir, ".cursor", "commands", "write.md")); err != nil {
		t.Fatalf("pipeline command should remain: %v", err)
	}
}

func TestCleanAll(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("build review exit = %d\n%s", code, stderr)
	}
	if _, stderr, code := runInProc("build", pipelineAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("build pipeline exit = %d\n%s", code, stderr)
	}
	if _, stderr, code := runInProc("clean", "--all", "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("clean --all exit = %d\n%s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(outDir, ".cursor", "commands", "write.md")); !os.IsNotExist(err) {
		t.Fatalf("pipeline command should be removed")
	}
	if _, err := os.Stat(filepath.Join(outDir, ".cursor", "commands", "ship.md")); !os.IsNotExist(err) {
		t.Fatalf("review command should be removed")
	}
}

func TestCleanDriftGuard(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("build exit = %d\n%s", code, stderr)
	}
	agentPath := filepath.Join(outDir, ".cursor", "agents", "reviewer.md")
	if err := os.WriteFile(agentPath, []byte("hand-edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runInProc("clean", reviewAf, "--target", "cursor", "--out", outDir)
	if code != 0 {
		t.Fatalf("clean with drift should still exit 0, got %d\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "AF310") {
		t.Fatalf("expected drift warning:\n%s", stderr)
	}
	if _, err := os.Stat(agentPath); err != nil {
		t.Fatalf("modified artifact should remain: %v", err)
	}
}

func TestCleanNoManifestExit1(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	_, stderr, code := runInProc("clean", reviewAf, "--target", "cursor", "--out", outDir)
	if code != 1 {
		t.Fatalf("clean without manifest exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "no manifest found") {
		t.Fatalf("expected no manifest error:\n%s", stderr)
	}
}

func TestBuildPruneRemovesOrphan(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("first build exit = %d\n%s", code, stderr)
	}

	orphanRel := ".cursor/agents/orphan.md"
	orphanFull := filepath.Join(outDir, filepath.FromSlash(orphanRel))
	if err := os.WriteFile(orphanFull, []byte("orphan"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPaths, err := filepath.Glob(filepath.Join(outDir, ".cursor", ".agentflow", "manifests", "*.json"))
	if err != nil || len(manifestPaths) != 1 {
		t.Fatalf("manifest glob: %v paths=%v", err, manifestPaths)
	}
	data, err := os.ReadFile(manifestPaths[0])
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	arts, _ := doc["artifacts"].([]any)
	arts = append(arts, map[string]any{
		"path":   orphanRel,
		"role":   "agent",
		"sha256": manifestHashBytes([]byte("orphan")),
		"bytes":  6,
	})
	doc["artifacts"] = arts
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPaths[0], updated, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir, "--prune"); code != 0 {
		t.Fatalf("second build --prune exit = %d\n%s", code, stderr)
	}
	if _, err := os.Stat(orphanFull); !os.IsNotExist(err) {
		t.Fatalf("orphan should be pruned")
	}
}

func TestVersionsListGrouped(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("build review exit = %d\n%s", code, stderr)
	}
	if _, stderr, code := runInProc("build", pipelineAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("build pipeline exit = %d\n%s", code, stderr)
	}
	out, _, code := runInProc("versions", "list", "--target", "cursor", "--out", outDir)
	if code != 0 {
		t.Fatalf("versions list exit = %d", code)
	}
	if !strings.Contains(out, "review.af") || !strings.Contains(out, "pipeline.af") {
		t.Fatalf("versions list should include both sources:\n%s", out)
	}
}

func TestVersionsStatusClean(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("build exit = %d\n%s", code, stderr)
	}
	out, _, code := runInProc("versions", "status", reviewAf, "--target", "cursor", "--out", outDir)
	if code != 0 {
		t.Fatalf("versions status exit = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "clean") {
		t.Fatalf("status should report clean artifacts:\n%s", out)
	}
}

func TestVersionsDiff(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("first build exit = %d\n%s", code, stderr)
	}
	if _, stderr, code := runInProc("build", reviewAf, "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("second build exit = %d\n%s", code, stderr)
	}
	out, _, code := runInProc("versions", "diff", reviewAf, "--target", "cursor", "--out", outDir)
	if code != 0 {
		t.Fatalf("versions diff exit = %d\n%s", code, out)
	}
	if !strings.Contains(out, "diff v1 -> v2") {
		t.Fatalf("expected default diff between latest builds:\n%s", out)
	}
}

func TestCleanCrossSourceSharedArtifact(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	sharedRel := ".cursor/agents/shared-agent.md"
	sharedFull := filepath.Join(outDir, filepath.FromSlash(sharedRel))
	if err := os.MkdirAll(filepath.Dir(sharedFull), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedFull, []byte("shared"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeCrossSourceManifest(t, outDir, "examples/a.af", []string{sharedRel, ".cursor/agents/a-only.md"})
	writeCrossSourceManifest(t, outDir, "examples/b.af", []string{sharedRel})

	if _, stderr, code := runInProc("clean", "examples/a.af", "--target", "cursor", "--out", outDir); code != 0 {
		t.Fatalf("clean a exit = %d\n%s", code, stderr)
	}
	if _, err := os.Stat(sharedFull); err != nil {
		t.Fatalf("shared artifact should remain while b still owns it: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, filepath.FromSlash(".cursor/agents/a-only.md"))); !os.IsNotExist(err) {
		t.Fatalf("a-only artifact should be removed")
	}
}

func writeCrossSourceManifest(t *testing.T, dir, source string, paths []string) {
	t.Helper()
	fs := emit.NewFS()
	for _, p := range paths {
		fs.Write(p, []byte("x"))
	}
	m := manifest.Build(manifest.BuildOptions{
		Target:       "cursor",
		SourcePath:   source,
		SourceSHA256: manifest.HashBytes([]byte(source)),
		IRHash:       "ir",
		FS:           fs,
		ToolVersion:  "test",
	})
	data, err := manifest.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	rel := manifest.ManifestRelPath("cursor", source)
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		artifactFull := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(artifactFull), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(artifactFull, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func manifestHashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

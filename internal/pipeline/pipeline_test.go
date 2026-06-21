package pipeline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/pipeline"
)

func examplesDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "..", "..", "examples")
}

func TestCompileReviewClean(t *testing.T) {
	t.Parallel()
	res, diags := pipeline.Compile(filepath.Join(examplesDir(t), "review.af"))
	if diags.HasErrors() {
		t.Fatalf("review.af should compile clean, got errors: %#v", diags)
	}
	if res.IR.Entry.FlowName != "ship" {
		t.Fatalf("entry flow = %q, want ship", res.IR.Entry.FlowName)
	}
	if res.IR.Entry.Trigger != "/ship" {
		t.Fatalf("entry trigger = %q, want /ship", res.IR.Entry.Trigger)
	}
	if res.Source == "" {
		t.Fatal("Result.Source should carry the original source for diag rendering")
	}
	if len(res.IR.Agents) == 0 {
		t.Fatal("expected agents in IR")
	}
}

func TestCompileSupplementaryFixturesClean(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"pipeline.af", "research.af", "critic.af", "docs.af", "cl-review.af"} {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, diags := pipeline.Compile(filepath.Join(examplesDir(t), name))
			if diags.HasErrors() {
				t.Fatalf("%s should validate clean, got errors: %#v", name, diags)
			}
		})
	}
}

func TestCompileReadErrorIsDiagnosticNotPanic(t *testing.T) {
	t.Parallel()
	res, diags := pipeline.Compile(filepath.Join(examplesDir(t), "does-not-exist.af"))
	if !diags.HasErrors() {
		t.Fatalf("missing file should yield an error diagnostic, got %#v", diags)
	}
	if len(diags) != 1 || diags[0].Code != "AF000" {
		t.Fatalf("want single AF000 read diagnostic, got %#v", diags)
	}
	if diags[0].Pos.Filename == "" {
		t.Fatal("read-error diagnostic should record the file path")
	}
	if len(res.IR.Agents) != 0 {
		t.Fatal("IR should be zero value on read error")
	}
}

func TestCompileParseErrorStopsPipeline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.af")
	if err := os.WriteFile(bad, []byte("this is not valid agentflow @@@\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, diags := pipeline.Compile(bad)
	if !diags.HasErrors() {
		t.Fatalf("garbage input should produce parse errors, got %#v", diags)
	}
}

package flowgraph

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/parser"
	"github.com/Abhinand20/agentFlow/internal/sema"
)

var update = flag.Bool("update", false, "rewrite golden resolved snapshots")

func TestGoldenResolved(t *testing.T) {
	for _, name := range []string{"review", "pipeline", "research", "critic", "docs"} {
		t.Run(name, func(t *testing.T) {
			_, file, _, _ := runtime.Caller(0)
			examplesDir := filepath.Join(filepath.Dir(file), "..", "..", "examples")
			src, err := os.ReadFile(filepath.Join(examplesDir, name+".af"))
			if err != nil {
				t.Fatal(err)
			}
			root, diags := parser.Parse(name+".af", string(src))
			if diags.HasErrors() {
				t.Fatalf("parse: %#v", diags)
			}
			prog, _ := sema.Resolve(root, examplesDir)
			res, resDiags := Resolve(prog)
			if resDiags.HasErrors() {
				t.Fatalf("resolve: %#v", resDiags)
			}
			got, err := MarshalSnapshot(res)
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", name+".resolved.json")
			if *update {
				os.MkdirAll("testdata", 0o755)
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run -update): %v", err)
			}
			if string(got) != string(want) {
				t.Fatalf("snapshot mismatch for %s", name)
			}
		})
	}
}

func TestReviewResolvedInvariants(t *testing.T) {
	res, _ := resolveFixture(t, "review")
	if res.Instances["code_review.build"] == nil {
		t.Fatal("missing inlined build")
	}
	cr := res.Instances["code_review"]
	if cr == nil || cr.ReturnsFrom != "code_review.reviewer" {
		t.Fatalf("code_review = %#v", cr)
	}
	rev := res.Instances["code_review.reviewer"]
	if rev == nil || len(rev.GatherPayload) != 3 {
		t.Fatalf("reviewer gather = %#v", rev)
	}
	for _, k := range []string{"lint", "security", "style"} {
		if rev.GatherPayload[k] == "" {
			t.Fatalf("missing gather key %q", k)
		}
	}
}

func TestPipelineResolvedInvariants(t *testing.T) {
	res, _ := resolveFixture(t, "pipeline")
	if res.Instances["edit"] == nil {
		t.Fatal("missing terminal edit")
	}
}

func TestCriticResolvedInvariants(t *testing.T) {
	res, _ := resolveFixture(t, "critic")
	if res.Instances["repeat.generate"] == nil || res.Instances["repeat.critic"] == nil {
		t.Fatalf("instances = %#v", res.Instances)
	}
	if res.Instances["repeat.critic"].ValueLabel != "verdict" {
		t.Fatalf("verdict = %q", res.Instances["repeat.critic"].ValueLabel)
	}
}

func TestResearchResolvedInvariants(t *testing.T) {
	res, _ := resolveFixture(t, "research")
	syn := res.Instances["synthesize"]
	if syn == nil || syn.ValueLabel != "report" {
		t.Fatalf("synthesize = %#v", syn)
	}
}

func TestDocsResolvedInvariants(t *testing.T) {
	res, _ := resolveFixture(t, "docs")
	if res.Instances["outline"] == nil || res.Instances["draft"] == nil {
		t.Fatalf("instances = %#v", res.Instances)
	}
	if res.Instances["draft"].Upstream != "outline" {
		t.Fatalf("draft upstream = %q", res.Instances["draft"].Upstream)
	}
}

package ir_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/ir"
)

var update = flag.Bool("update", false, "rewrite golden IR snapshots")

func TestGoldenIR(t *testing.T) {
	for _, name := range []string{"review", "pipeline", "research", "critic", "docs"} {
		t.Run(name, func(t *testing.T) {
			p, _, _ := compileIR(t, name)
			got, err := ir.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", name+".ir.json")
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
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

package render_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/ir"
	"github.com/Abhinand20/agentFlow/internal/render"
)

func loadIRFixture(t *testing.T, name string) ir.Program {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "ir", "testdata", name+".ir.json"))
	if err != nil {
		t.Fatalf("read IR fixture %s: %v", name, err)
	}
	p, err := ir.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("unmarshal IR fixture %s: %v", name, err)
	}
	return p
}

func findAgent(t *testing.T, p ir.Program, name string) ir.Agent {
	t.Helper()
	for _, a := range p.Agents {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("agent not found: %s", name)
	return ir.Agent{}
}

func runbookOrFatal(t *testing.T, p ir.Program, v render.Vocabulary) string {
	t.Helper()
	got, err := render.Runbook(p, v)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

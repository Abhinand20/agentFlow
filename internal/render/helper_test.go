package render_test

import (
	"os"
	"path/filepath"

	"github.com/Abhinand20/agentFlow/internal/ir"
)

func loadIRFixture(name string) ir.Program {
	data, err := os.ReadFile(filepath.Join("..", "ir", "testdata", name+".ir.json"))
	if err != nil {
		panic(err)
	}
	p, err := ir.UnmarshalJSON(data)
	if err != nil {
		panic(err)
	}
	return p
}

func findAgent(p ir.Program, name string) ir.Agent {
	for _, a := range p.Agents {
		if a.Name == name {
			return a
		}
	}
	panic("agent not found: " + name)
}

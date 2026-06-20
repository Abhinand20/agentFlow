package binding_test

import (
	"testing"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/emit"
	"github.com/Abhinand20/agentFlow/internal/ir"
)

type stubBinding struct{}

func (stubBinding) Name() string { return "stub" }

func (stubBinding) Capabilities() map[binding.Capability]bool { return nil }

func (stubBinding) Emit(_ ir.Program) (*emit.FS, diag.Diagnostics) {
	return emit.NewFS(), nil
}

func TestRegisterGet(t *testing.T) {
	b := stubBinding{}
	binding.Register(b)

	got, ok := binding.Get("stub")
	if !ok {
		t.Fatal("expected stub binding to be registered")
	}
	if got.Name() != "stub" {
		t.Fatalf("Name() = %q, want stub", got.Name())
	}
}

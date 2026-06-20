package ir_test

import (
	"bytes"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/ir"
)

func TestMarshalIdempotent(t *testing.T) {
	p, _, _ := compileIR(t, "review")
	b1, err := ir.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := ir.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("marshal twice produced different bytes")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	p, _, _ := compileIR(t, "review")
	b1, err := ir.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := ir.UnmarshalJSON(b1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := ir.Marshal(p2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("round-trip marshal produced different bytes")
	}
}

func TestMarshalPipelineRoundTrip(t *testing.T) {
	p, _, _ := compileIR(t, "pipeline")
	b1, err := ir.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := ir.UnmarshalJSON(b1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := ir.Marshal(p2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("round-trip marshal produced different bytes")
	}
}

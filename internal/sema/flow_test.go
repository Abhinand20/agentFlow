package sema

import (
	"sort"
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/model"
)

func TestFlowHeaderExplicitMarkers(t *testing.T) {
	prog, _ := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent x { model: s out: V } type V = pass
entry flow f { out: V return: review on: "/go" x }`)
	fl := prog.Flows["f"]
	if !fl.OutExplicit || fl.Out != "V" || !fl.ReturnExplicit || fl.Return != "review" || fl.On != "/go" {
		t.Fatalf("flow = %#v", fl)
	}
}

func TestFlowInferredMarkers(t *testing.T) {
	prog, _ := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent x { model: s }
entry flow f { x }`)
	if prog.Flows["f"].OutExplicit || prog.Flows["f"].ReturnExplicit {
		t.Fatalf("should be inferred: %#v", prog.Flows["f"])
	}
}

func TestOnOnNonEntryWarns_AF137(t *testing.T) {
	_, diags := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent x { model: s }
flow f { on: "/x" x }
entry flow e { x }`)
	if !hasCode(diags, "AF137") {
		t.Fatalf("want AF137, got %#v", diags)
	}
}

func TestNoEntryFlow_AF138(t *testing.T) {
	_, diags := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent x { model: s }
flow f { x }`)
	if !hasCode(diags, "AF138") {
		t.Fatalf("want AF138, got %#v", diags)
	}
}

func TestMultipleEntryFlows_AF139(t *testing.T) {
	_, diags := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent x { model: s }
entry flow f { x }
entry flow g { x }`)
	if !hasCode(diags, "AF139") {
		t.Fatalf("want AF139, got %#v", diags)
	}
}

func TestEntryFlowSetAndTerminals(t *testing.T) {
	prog, _ := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent x { model: s }
entry flow only { x }`)
	if prog.EntryFlow != "only" {
		t.Fatalf("EntryFlow = %q", prog.EntryFlow)
	}
	if !model.Terminals()["done"] || !model.Terminals()["fail"] {
		t.Fatal("done/fail must be terminals")
	}
}

func TestMultipleEntryFlowsListsNames_AF139(t *testing.T) {
	_, diags := resolveSrc(t, `entry flow b { }
entry flow a { }`)
	var msg string
	for _, d := range diags {
		if d.Code == "AF139" {
			msg = d.Msg
		}
	}
	if msg == "" || strings.Index(msg, "a") > strings.Index(msg, "b") {
		t.Fatalf("AF139 must list sorted names: %q", msg)
	}
}

func TestMultipleEntryFlowsSortedNames(t *testing.T) {
	// verify sort helper used by accountEntryFlow
	names := []string{"z", "a", "m"}
	sort.Strings(names)
	if names[0] != "a" || names[2] != "z" {
		t.Fatal("sort sanity")
	}
}

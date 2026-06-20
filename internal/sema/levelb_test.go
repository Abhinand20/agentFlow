package sema

import "testing"

func TestLevelBFlowParams_AF150(t *testing.T) {
	_, diags := resolveSrc(t, `flow f(a, b: T) { done }`)
	if !hasCode(diags, "AF150") {
		t.Fatalf("want AF150 for params, got %#v", diags)
	}
}

func TestLevelBCall_AF150(t *testing.T) {
	_, diags := resolveSrc(t, `flow f { summarize(topic) { done } }`)
	if !hasCode(diags, "AF150") {
		t.Fatalf("want AF150 for call, got %#v", diags)
	}
}

func TestLevelBParallelEach_AF150(t *testing.T) {
	_, diags := resolveSrc(t, `flow f { parallel each items as x { done } }`)
	if !hasCode(diags, "AF150") {
		t.Fatalf("want AF150 for parallel each, got %#v", diags)
	}
}

func TestLevelBBranchIt_AF150(t *testing.T) {
	_, diags := resolveSrc(t, `flow f { branch it { case a -> done } }`)
	if !hasCode(diags, "AF150") {
		t.Fatalf("want AF150 for it, got %#v", diags)
	}
}

func TestLevelBUseAlias_AF150(t *testing.T) {
	_, diags := resolveSrc(t, `use std.patterns as p`)
	if !hasCode(diags, "AF150") {
		t.Fatalf("want AF150 for use-as, got %#v", diags)
	}
}

func TestLevelBEdgeAttr_AF150(t *testing.T) {
	_, diags := resolveSrc(t, `flow f { a -> b [when v == x] }`)
	if !hasCode(diags, "AF150") {
		t.Fatalf("want AF150 for edge attr, got %#v", diags)
	}
}

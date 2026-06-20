package sema

import (
	"strings"
	"testing"
)

func TestModelResolveUnqualifiedUnique(t *testing.T) {
	prog, diags := resolveSrc(t, `use anthropic { kind: model-provider models: [opus, sonnet] }
agent a { model: sonnet }`)
	if diags.HasErrors() {
		t.Fatalf("unexpected: %#v", diags)
	}
	ag := prog.Agents["a"]
	if ag.ModelProvider != "anthropic" || ag.ResolvedAlias != "sonnet" {
		t.Fatalf("resolved = %#v", ag)
	}
}

func TestModelResolveUnknownAlias_AF110(t *testing.T) {
	_, diags := resolveSrc(t, `use anthropic { kind: model-provider models: [opus] }
agent a { model: gpt }`)
	if !hasCode(diags, "AF110") {
		t.Fatalf("want AF110, got %#v", diags)
	}
}

func TestModelResolveAmbiguousListsCandidatesSorted_AF110(t *testing.T) {
	_, diags := resolveSrc(t, `use zeta { kind: model-provider models: [opus] }
use alpha { kind: model-provider models: [opus] }
agent a { model: opus }`)
	var msg string
	for _, d := range diags {
		if d.Code == "AF110" {
			msg = d.Msg
		}
	}
	if msg == "" || strings.Index(msg, "alpha") > strings.Index(msg, "zeta") {
		t.Fatalf("AF110 must list candidates sorted: %q", msg)
	}
}

func TestModelResolveQualifiedMissingAlias_AF110(t *testing.T) {
	_, diags := resolveSrc(t, `use anthropic { kind: model-provider models: [opus] }
agent a { model: anthropic.gpt }`)
	if !hasCode(diags, "AF110") {
		t.Fatalf("want AF110, got %#v", diags)
	}
}

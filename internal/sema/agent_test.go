package sema

import "testing"

func TestAgentFieldsParsed(t *testing.T) {
	prog, _ := resolveSrc(t, `use a { kind: model-provider models: [opus] }
agent r { model: opus out: Verdict tools: [github.get_pr] permissions: "read" retry: 1 description: "d" }`)
	ag := prog.Agents["r"]
	if ag.ModelAlias != "opus" || ag.Out != "Verdict" || ag.Permissions != "read" || ag.Retry != 1 {
		t.Fatalf("agent = %#v", ag)
	}
	if len(ag.Tools) != 1 || ag.Tools[0].Capability != "github" || ag.Tools[0].Tool != "get_pr" {
		t.Fatalf("tools = %#v", ag.Tools)
	}
}

func TestAgentMissingModel_AF132(t *testing.T) {
	_, diags := resolveSrc(t, `agent r { prompt: "hi" }`)
	if !hasCode(diags, "AF132") {
		t.Fatalf("want AF132, got %#v", diags)
	}
}

func TestAgentNonIntegerRetry_AF000(t *testing.T) {
	_, diags := resolveSrc(t, `agent r { model: opus retry: 1.5 }`)
	if !hasCode(diags, "AF000") {
		t.Fatalf("want AF000, got %#v", diags)
	}
}

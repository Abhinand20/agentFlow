package sema

import "testing"

func TestCapabilityModelProviderParsed(t *testing.T) {
	prog, diags := resolveSrc(t, `use anthropic { kind: model-provider models: [opus, sonnet] }`)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %#v", diags)
	}
	c := prog.Capabilities["anthropic"]
	if c.Kind != "model-provider" || len(c.Models) != 2 || c.Models[0] != "opus" {
		t.Fatalf("capability = %#v", c)
	}
}

func TestCapabilityMcpRequiresCommand_AF130(t *testing.T) {
	_, diags := resolveSrc(t, `use gh { kind: mcp transport: stdio }`)
	if !hasCode(diags, "AF130") {
		t.Fatalf("want AF130, got %#v", diags)
	}
}

func TestCapabilityModelProviderRequiresModels_AF131(t *testing.T) {
	_, diags := resolveSrc(t, `use x { kind: model-provider }`)
	if !hasCode(diags, "AF131") {
		t.Fatalf("want AF131, got %#v", diags)
	}
}

func TestUnknownFieldWarnsAndRetained_AF120(t *testing.T) {
	prog, diags := resolveSrc(t, `use x { kind: model-provider models: [opus] foo: bar }`)
	if !hasCode(diags, "AF120") {
		t.Fatalf("want AF120, got %#v", diags)
	}
	if prog.Capabilities["x"].Raw["foo"] == nil {
		t.Fatalf("unknown field must be retained in Raw")
	}
}

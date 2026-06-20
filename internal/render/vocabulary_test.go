package render_test

import (
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/render"
)

func TestDefaultVocabulary(t *testing.T) {
	v := render.DefaultVocabulary()

	agent := render.AgentView{
		Decl:         "reviewer",
		PrevProducer: "build",
	}
	got := v.InvokeAgent(agent)
	if !strings.Contains(got, "`reviewer`") || !strings.Contains(got, "`build`") {
		t.Fatalf("InvokeAgent: %q", got)
	}

	gate := render.GateView{Run: "scripts/test.sh"}
	if v.RunScript(gate) != "Run `scripts/test.sh`." {
		t.Fatalf("RunScript: %q", v.RunScript(gate))
	}

	branches := []render.StepView{
		{Decl: "lint"},
		{Decl: "security"},
	}
	got = v.SpawnParallel(branches)
	if !strings.Contains(got, "lint") || !strings.Contains(got, "security") {
		t.Fatalf("SpawnParallel: %q", got)
	}

	if v.ReadOutput("review") != "Read the `out:` value from `review`." {
		t.Fatalf("ReadOutput: %q", v.ReadOutput("review"))
	}

	parse := v.ParseOutputProtocol([]string{"approve", "reject"}, 2)
	if !strings.Contains(parse, "2") || !strings.Contains(parse, "approve") {
		t.Fatalf("ParseOutputProtocol: %q", parse)
	}

	if v.GotoStep("build") != "Go back to step `build`." {
		t.Fatalf("GotoStep: %q", v.GotoStep("build"))
	}

	if v.Arg("input") != "<input>" {
		t.Fatalf("Arg: %q", v.Arg("input"))
	}
}

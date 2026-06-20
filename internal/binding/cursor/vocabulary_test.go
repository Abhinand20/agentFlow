package cursor_test

import (
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/binding/cursor"
	"github.com/Abhinand20/agentFlow/internal/render"
)

func TestCursorVocabulary(t *testing.T) {
	v := cursor.Vocabulary()

	t.Run("InvokeAgent", func(t *testing.T) {
		got := v.InvokeAgent(render.AgentView{Decl: "reviewer"})
		if !strings.Contains(got, "`reviewer`") || !strings.Contains(got, "rule") {
			t.Fatalf("InvokeAgent basic: %q", got)
		}

		got = v.InvokeAgent(render.AgentView{
			Decl:         "build",
			UsesFlowArg:  true,
			PrevProducer: "ticket",
		})
		if !strings.Contains(got, "$1") {
			t.Fatalf("InvokeAgent flow arg: %q", got)
		}
		if !strings.Contains(got, "`ticket`") {
			t.Fatalf("InvokeAgent prev producer: %q", got)
		}
	})

	t.Run("RunScript", func(t *testing.T) {
		got := v.RunScript(render.GateView{Run: "scripts/test.sh"})
		want := "Run `scripts/test.sh` in the terminal."
		if got != want {
			t.Fatalf("RunScript = %q, want %q", got, want)
		}
	})

	t.Run("SpawnParallel", func(t *testing.T) {
		branches := []render.StepView{
			{Decl: "lint"},
			{Decl: "security"},
		}
		got := v.SpawnParallel(branches)
		if !strings.Contains(got, "one after another") {
			t.Fatalf("SpawnParallel missing sequential fallback: %q", got)
		}
		if !strings.Contains(got, "lint") || !strings.Contains(got, "security") {
			t.Fatalf("SpawnParallel missing branch names: %q", got)
		}
	})

	t.Run("ReadOutput", func(t *testing.T) {
		got := v.ReadOutput("review")
		want := "Read the `out:` value from `review`."
		if got != want {
			t.Fatalf("ReadOutput = %q, want %q", got, want)
		}
	})

	t.Run("ParseOutputProtocol", func(t *testing.T) {
		got := v.ParseOutputProtocol([]string{"approve", "reject"}, 2)
		if !strings.Contains(got, "2") || !strings.Contains(got, "approve") {
			t.Fatalf("ParseOutputProtocol: %q", got)
		}
	})

	t.Run("GotoStep", func(t *testing.T) {
		got := v.GotoStep("build")
		want := "Go back to step `build`."
		if got != want {
			t.Fatalf("GotoStep = %q, want %q", got, want)
		}
	})

	t.Run("Arg", func(t *testing.T) {
		if v.Arg("input") != "$1" {
			t.Fatalf("Arg = %q, want $1", v.Arg("input"))
		}
	})
}

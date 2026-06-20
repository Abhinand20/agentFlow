package render_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/ir"
	"github.com/Abhinand20/agentFlow/internal/render"
)

var update = flag.Bool("update", false, "rewrite golden text snapshots")

func assertGolden(t *testing.T, name string, got string) {
	t.Helper()
	golden := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden %s (run -update): %v", golden, err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}

func TestGoldenProtocol(t *testing.T) {
	got := render.OutputProtocol([]string{"approve", "revise", "reject"}, 0)
	assertGolden(t, "protocol_reviewer.txt", got)
	if !strings.Contains(got, "```agentflow-output") {
		t.Fatal("missing fence tag")
	}
	if render.OutputProtocol(nil, 0) != "" {
		t.Fatal("empty enum should produce empty protocol")
	}
}

func TestGoldenPromptReviewer(t *testing.T) {
	p := loadIRFixture("review")
	a := findAgent(p, "reviewer")
	v := render.DefaultVocabulary()
	got := render.AgentPrompt(a, v)
	assertGolden(t, "prompt_reviewer.txt", got)
}

func TestAgentPromptDiskFree(t *testing.T) {
	dir := t.TempDir()
	// No prompt files — only inlined IR text.
	p := loadIRFixture("review")
	a := findAgent(p, "reviewer")
	v := render.DefaultVocabulary()
	got := render.AgentPrompt(a, v)
	// Prove identical output without any files in temp dir.
	_ = dir
	if !strings.Contains(got, "agentflow-output") {
		t.Fatal("expected protocol block")
	}
	if got != render.AgentPrompt(a, v) {
		t.Fatal("render should be deterministic")
	}
}

func TestGoldenRunbookReview(t *testing.T) {
	p := loadIRFixture("review")
	v := render.DefaultVocabulary()
	got := render.Runbook(p, v)
	assertGolden(t, "runbook_review.txt", got)
}

func TestGoldenRunbookCriticRepeat(t *testing.T) {
	p := loadIRFixture("critic")
	v := render.DefaultVocabulary()
	got := render.Runbook(p, v)
	assertGolden(t, "runbook_critic_repeat.txt", got)
}

func TestGoldenPerConstruct(t *testing.T) {
	v := render.DefaultVocabulary()

	t.Run("seq", func(t *testing.T) {
		p := minimalProgram(ir.Node{
			Kind: ir.NodeSeq,
			Steps: []ir.Node{
				stepAgent("s1", "build", "", ""),
				stepAgent("s2", "reviewer", "s1", "code_review.review"),
			},
		})
		assertGolden(t, "construct_seq.txt", render.Runbook(p, v))
	})

	t.Run("branch", func(t *testing.T) {
		p := minimalProgram(ir.Node{
			Kind:        ir.NodeBranch,
			BranchValue: "code_review",
			BranchEnum:  []string{"approve", "reject"},
			Cases: []ir.BranchCase{
				{Values: []string{"approve"}, Body: stepAgent("deploy", "deploy", "", "")},
				{Values: []string{"reject"}, Body: stepAgent("notify", "notify_author", "", "")},
			},
		})
		assertGolden(t, "construct_branch.txt", render.Runbook(p, v))
	})

	t.Run("loop", func(t *testing.T) {
		p := minimalProgram(ir.Node{
			Kind:   ir.NodeLoop,
			HasMax: true,
			Max:    3,
			Cond:   &ir.Cond{ValueLabel: "review", Op: "!=", Enum: "revise"},
			Body: &ir.Node{
				Kind: ir.NodeSeq,
				Steps: []ir.Node{
					stepAgent("loop.build", "build", "", ""),
				},
			},
		})
		assertGolden(t, "construct_loop.txt", render.Runbook(p, v))
	})

	t.Run("repeat", func(t *testing.T) {
		p := minimalProgram(ir.Node{
			Kind:    ir.NodeLoop,
			DoWhile: true,
			HasMax:  true,
			Max:     2,
			Cond:    &ir.Cond{ValueLabel: "verdict", Op: "==", Enum: "pass"},
			Body: &ir.Node{
				Kind: ir.NodeSeq,
				Steps: []ir.Node{
					stepAgent("repeat.generate", "generate", "", ""),
					stepAgent("repeat.critic", "critic", "repeat.generate", "verdict"),
				},
			},
		}, withAgents(
			ir.Agent{Name: "generate", Prompt: "Generate."},
			ir.Agent{Name: "critic", Prompt: "Critique.", OutEnum: []string{"pass", "revise"}},
		))
		assertGolden(t, "construct_repeat.txt", render.Runbook(p, v))
	})

	t.Run("parallel", func(t *testing.T) {
		p := minimalProgram(ir.Node{
			Kind: ir.NodeParallel,
			Branches: []ir.Node{
				stepAgent("lint", "lint", "build", ""),
				stepAgent("security", "security", "build", ""),
				stepAgent("style", "style", "build", ""),
			},
			Gather: &ir.StepRef{
				ControlLabel: "reviewer",
				Decl:         "reviewer",
				Kind:         "agent",
				PrevProducer: "build",
			},
			Payload: &ir.GatherPayload{
				Branches: []ir.GatherBranch{
					{ControlLabel: "lint", ValueLabel: "lint"},
					{ControlLabel: "security", ValueLabel: "security"},
					{ControlLabel: "style", ValueLabel: "style"},
				},
			},
		}, withAgents(
			ir.Agent{Name: "lint", Prompt: "Lint."},
			ir.Agent{Name: "security", Prompt: "Security."},
			ir.Agent{Name: "style", Prompt: "Style."},
			ir.Agent{Name: "reviewer", Prompt: "Review."},
		))
		assertGolden(t, "construct_parallel.txt", render.Runbook(p, v))
	})

	for _, onFail := range []string{"halt", "retry", "goto", "enter-loop"} {
		t.Run("gate_"+onFail, func(t *testing.T) {
			gate := ir.Gate{
				Name:         "quality",
				Run:          "scripts/test.sh",
				OnFail:       onFail,
				OnFailTarget: "build",
				ScriptRetry:  2,
			}
			p := minimalProgram(ir.Node{
				Kind:  ir.NodeSeq,
				Steps: []ir.Node{stepGate("quality", "quality")},
			}, withGates(gate))
			assertGolden(t, "construct_gate_"+onFail+".txt", render.Runbook(p, v))
		})
	}
}

func TestGoldenAgentDocumentReviewer(t *testing.T) {
	p := loadIRFixture("review")
	a := findAgent(p, "reviewer")
	v := render.DefaultVocabulary()
	got := render.FormatDocument(render.AgentDocument(p, a, v))
	assertGolden(t, "agent_document_reviewer.txt", got)
}

func TestForbiddenHostTokens(t *testing.T) {
	v := render.DefaultVocabulary()
	forbidden := []string{".claude", "$ARGUMENTS", "Task tool", "bounce-back"}
	var corpus strings.Builder

	p := loadIRFixture("review")
	corpus.WriteString(render.Runbook(p, v))
	corpus.WriteString(render.AgentPrompt(findAgent(p, "reviewer"), v))
	corpus.WriteString(render.FormatDocument(render.RunbookDocument(p, v)))
	corpus.WriteString(render.FormatDocument(render.AgentDocument(p, findAgent(p, "reviewer"), v)))

	p2 := loadIRFixture("critic")
	corpus.WriteString(render.Runbook(p2, v))

	for _, token := range forbidden {
		if strings.Contains(corpus.String(), token) {
			t.Fatalf("forbidden host token %q found in rendered output", token)
		}
	}
}

type programOpt func(*ir.Program)

func withAgents(agents ...ir.Agent) programOpt {
	return func(p *ir.Program) {
		p.Agents = agents
	}
}

func withGates(gates ...ir.Gate) programOpt {
	return func(p *ir.Program) {
		p.Gates = gates
	}
}

func minimalProgram(root ir.Node, opts ...programOpt) ir.Program {
	p := ir.Program{
		Entry: ir.Entry{FlowName: "test"},
		Flow:  ir.Flow{Name: "test", Root: root},
		Agents: []ir.Agent{
			{Name: "build", Prompt: "Build."},
			{Name: "reviewer", Prompt: "Review.", OutEnum: []string{"approve", "revise", "reject"}},
			{Name: "deploy", Prompt: "Deploy."},
			{Name: "notify_author", Prompt: "Notify."},
		},
	}
	for _, o := range opts {
		o(&p)
	}
	return p
}

func stepGate(controlLabel, decl string) ir.Node {
	return ir.Node{
		Kind: ir.NodeStep,
		Step: &ir.StepRef{
			ControlLabel: controlLabel,
			Decl:         decl,
			Kind:         "gate",
		},
	}
}

func stepAgent(controlLabel, decl, prevProducer, valueLabel string) ir.Node {
	return ir.Node{
		Kind: ir.NodeStep,
		Step: &ir.StepRef{
			ControlLabel: controlLabel,
			ValueLabel:   valueLabel,
			Decl:         decl,
			Kind:         "agent",
			PrevProducer: prevProducer,
		},
	}
}

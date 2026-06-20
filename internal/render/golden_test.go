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
	p := loadIRFixture(t, "review")
	a := findAgent(t, p, "reviewer")
	got := render.AgentPrompt(a)
	assertGolden(t, "prompt_reviewer.txt", got)
}

func TestAgentPromptDiskFree(t *testing.T) {
	p := loadIRFixture(t, "review")
	a := findAgent(t, p, "reviewer")
	expected := render.AgentPrompt(a)

	dir := t.TempDir()
	fakePath := filepath.Join(dir, "prompts", "reviewer.md")
	if err := os.MkdirAll(filepath.Dir(fakePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fakePath, []byte("WRONG CONTENT FROM DISK"), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("restore wd: %v", err)
		}
	}()

	if err := os.Remove(fakePath); err != nil {
		t.Fatal(err)
	}

	got := render.AgentPrompt(a)
	if got != expected {
		t.Fatalf("disk-free render mismatch\n--- got ---\n%s\n--- want ---\n%s", got, expected)
	}
	if strings.Contains(got, "WRONG CONTENT FROM DISK") {
		t.Fatal("render read deleted prompt file from disk")
	}
}

func TestGoldenRunbookReview(t *testing.T) {
	p := loadIRFixture(t, "review")
	v := render.DefaultVocabulary()
	assertGolden(t, "runbook_review.txt", runbookOrFatal(t, p, v))
}

func TestGoldenRunbookCriticRepeat(t *testing.T) {
	p := loadIRFixture(t, "critic")
	v := render.DefaultVocabulary()
	assertGolden(t, "runbook_critic_repeat.txt", runbookOrFatal(t, p, v))
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
		assertGolden(t, "construct_seq.txt", runbookOrFatal(t, p, v))
	})

	t.Run("flow_arg", func(t *testing.T) {
		p := minimalProgram(ir.Node{
			Kind:  ir.NodeSeq,
			Steps: []ir.Node{stepAgent("s1", "build", "", "")},
		})
		p.Entry.InType = "Ticket"
		p.Agents[0] = ir.Agent{Name: "build", Prompt: "Build.", In: "Ticket"}
		assertGolden(t, "construct_flow_arg.txt", runbookOrFatal(t, p, v))
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
		assertGolden(t, "construct_branch.txt", runbookOrFatal(t, p, v))
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
		assertGolden(t, "construct_loop.txt", runbookOrFatal(t, p, v))
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
					stepAgentWithOutEnum("repeat.critic", "critic", "repeat.generate", "verdict", []string{"pass", "revise"}),
				},
			},
		}, withAgents(
			ir.Agent{Name: "generate", Prompt: "Generate."},
			ir.Agent{Name: "critic", Prompt: "Critique.", OutEnum: []string{"pass", "revise"}},
		))
		assertGolden(t, "construct_repeat.txt", runbookOrFatal(t, p, v))
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
		assertGolden(t, "construct_parallel.txt", runbookOrFatal(t, p, v))
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
			assertGolden(t, "construct_gate_"+onFail+".txt", runbookOrFatal(t, p, v))
		})
	}
}

func TestRunbookMissingAgent(t *testing.T) {
	p := minimalProgram(stepAgent("s1", "missing_agent", "", ""))
	_, err := render.Runbook(p, render.DefaultVocabulary())
	if err == nil {
		t.Fatal("expected error for missing agent decl")
	}
}

func TestRunbookMissingGate(t *testing.T) {
	p := minimalProgram(ir.Node{
		Kind:  ir.NodeSeq,
		Steps: []ir.Node{stepGate("quality", "quality")},
	})
	_, err := render.Runbook(p, render.DefaultVocabulary())
	if err == nil {
		t.Fatal("expected error for missing gate decl")
	}
}

func TestGoldenAgentDocumentReviewer(t *testing.T) {
	p := loadIRFixture(t, "review")
	a := findAgent(t, p, "reviewer")
	v := render.DefaultVocabulary()
	got := render.FormatDocument(render.AgentDocument(a, v))
	assertGolden(t, "agent_document_reviewer.txt", got)
}

func TestForbiddenHostTokens(t *testing.T) {
	v := render.DefaultVocabulary()
	forbidden := []string{".claude", "$ARGUMENTS", "Task tool", "bounce-back"}
	var corpus strings.Builder

	p := loadIRFixture(t, "review")
	corpus.WriteString(runbookOrFatal(t, p, v))
	corpus.WriteString(render.AgentPrompt(findAgent(t, p, "reviewer")))
	doc, err := render.RunbookDocument(p, v)
	if err != nil {
		t.Fatal(err)
	}
	corpus.WriteString(render.FormatDocument(doc))
	corpus.WriteString(render.FormatDocument(render.AgentDocument(findAgent(t, p, "reviewer"), v)))

	p2 := loadIRFixture(t, "critic")
	corpus.WriteString(runbookOrFatal(t, p2, v))

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

func stepAgentWithOutEnum(controlLabel, decl, prevProducer, valueLabel string, outEnum []string) ir.Node {
	n := stepAgent(controlLabel, decl, prevProducer, valueLabel)
	n.Step.OutEnum = outEnum
	return n
}

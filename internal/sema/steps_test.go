package sema

import (
	"testing"

	"github.com/Abhinand20/agentFlow/internal/model"
)

func TestResolveChainSteps(t *testing.T) {
	prog, _ := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent research { model: s } agent write { model: s }
entry flow f { research -> write }`)
	body := prog.Flows["f"].Body
	if len(body) != 1 || body[0].Kind != model.StepChain || len(body[0].Chain) != 2 {
		t.Fatalf("body = %#v", body)
	}
	if body[0].Chain[0].Ref != "research" || body[0].Chain[1].Ref != "write" {
		t.Fatalf("chain = %#v", body[0].Chain)
	}
}

func TestResolveRepeatDoWhileWithMax(t *testing.T) {
	prog, _ := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent g { model: s } agent c { model: s out: V }
type V = pass | revise
entry flow f { repeat { g as draft c as verdict } until (verdict == pass, max 3) }`)
	st := prog.Flows["f"].Body[0]
	if st.Kind != model.StepRepeat || !st.DoWhile || st.Max == nil || *st.Max != 3 {
		t.Fatalf("repeat = %#v", st)
	}
	if st.Cond == nil || st.Cond.Value != "verdict" || st.Cond.Op != "==" || st.Cond.Enum != "pass" {
		t.Fatalf("cond = %#v", st.Cond)
	}
}

func TestResolveBranchCases(t *testing.T) {
	prog, _ := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent x { model: s } agent y { model: s }
entry flow f { branch code_review { case approve -> x
case reject -> y } }`)
	st := prog.Flows["f"].Body[0]
	if st.Kind != model.StepBranch || st.BranchValue != "code_review" || len(st.Cases) != 2 {
		t.Fatalf("branch = %#v", st)
	}
}

func TestResolveMaxNonInteger_AF000(t *testing.T) {
	_, diags := resolveSrc(t, `use a { kind: model-provider models: [s] }
agent g { model: s }
entry flow f { loop (max 3.5) { g } }`)
	if !hasCode(diags, "AF000") {
		t.Fatalf("want AF000 for non-integer max, got %#v", diags)
	}
}

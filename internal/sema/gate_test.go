package sema

import (
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/model"
)

func TestGateRetryWithTargetAndBlocking(t *testing.T) {
	prog, diags := resolveSrc(t, `gate q { run: "t.sh" on-fail: retry on-fail-target: build behavior: blocking retry: 2 }`)
	if diags.HasErrors() {
		t.Fatalf("unexpected: %#v", diags)
	}
	g := prog.Gates["q"]
	if g.OnFail != model.FailRetryStep || g.OnFailTarget != "build" || g.Behavior != "blocking" || g.ScriptRetry != 2 {
		t.Fatalf("gate = %#v", g)
	}
}

func TestGateBehaviorDefaultsAdvisory(t *testing.T) {
	prog, _ := resolveSrc(t, `gate q { run: "t.sh" }`)
	if prog.Gates["q"].Behavior != "advisory" || prog.Gates["q"].OnFail != model.FailHalt {
		t.Fatalf("gate = %#v", prog.Gates["q"])
	}
}

func TestGateBounceBack_AF134(t *testing.T) {
	_, diags := resolveSrc(t, `gate q { run: "t.sh" on-fail: bounce-back }`)
	if !hasCode(diags, "AF134") {
		t.Fatalf("want AF134, got %#v", diags)
	}
	for _, d := range diags {
		if d.Code == "AF134" && !strings.Contains(d.Msg, "bounce-back") {
			t.Fatalf("want bounce-back message, got %q", d.Msg)
		}
	}
}

func TestGateInvalidOnFail_AF134(t *testing.T) {
	_, diags := resolveSrc(t, `gate q { run: "t.sh" on-fail: explode }`)
	if !hasCode(diags, "AF134") {
		t.Fatalf("want AF134, got %#v", diags)
	}
	for _, d := range diags {
		if d.Code == "AF134" && !strings.Contains(d.Msg, "invalid on-fail action") {
			t.Fatalf("want generic invalid message, got %q", d.Msg)
		}
	}
}

func TestGateRetryNeedsTarget_AF135(t *testing.T) {
	_, diags := resolveSrc(t, `gate q { run: "t.sh" on-fail: retry }`)
	if !hasCode(diags, "AF135") {
		t.Fatalf("want AF135, got %#v", diags)
	}
}

func TestGateGotoNeedsTarget_AF135(t *testing.T) {
	_, diags := resolveSrc(t, `gate q { run: "t.sh" on-fail: goto }`)
	if !hasCode(diags, "AF135") {
		t.Fatalf("want AF135, got %#v", diags)
	}
}

func TestGateHaltWithTargetWarns_AF136(t *testing.T) {
	_, diags := resolveSrc(t, `gate q { run: "t.sh" on-fail: halt on-fail-target: build }`)
	if !hasCode(diags, "AF136") {
		t.Fatalf("want AF136, got %#v", diags)
	}
}

package sema

import "testing"

func TestEnumTypeValues(t *testing.T) {
	prog, _ := resolveSrc(t, `type Verdict = approve | revise | reject`)
	if got := prog.Types["Verdict"].Values; len(got) != 3 || got[1] != "revise" {
		t.Fatalf("values = %#v", got)
	}
}

func TestEnumDuplicateMemberWarns_AF133(t *testing.T) {
	prog, diags := resolveSrc(t, `type T = a | a | b`)
	if !hasCode(diags, "AF133") {
		t.Fatalf("want AF133, got %#v", diags)
	}
	if len(prog.Types["T"].Values) != 2 {
		t.Fatalf("values should be dedup'd: %#v", prog.Types["T"].Values)
	}
}

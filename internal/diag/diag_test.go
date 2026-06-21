package diag_test

import (
	"testing"

	"github.com/alecthomas/participle/v2/lexer"

	"github.com/Abhinand20/agentFlow/internal/diag"
)

func TestDiagnosticsHasErrors(t *testing.T) {
	t.Parallel()

	warnOnly := diag.Diagnostics{
		{Severity: diag.Warning, Code: "AF204", Msg: "non-exhaustive branch"},
	}
	if warnOnly.HasErrors() {
		t.Fatal("expected no errors from warnings only")
	}

	withErr := diag.Diagnostics{
		{Severity: diag.Warning, Code: "AF204", Msg: "non-exhaustive branch"},
		{Severity: diag.Error, Code: "AF000", Msg: "unexpected token"},
	}
	if !withErr.HasErrors() {
		t.Fatal("expected HasErrors true when an error is present")
	}
}

func TestDiagnosticsAdd(t *testing.T) {
	t.Parallel()

	var diags diag.Diagnostics
	diags.Add(diag.Diagnostic{Code: "AF000", Severity: diag.Error, Msg: "first"})
	diags.Add(diag.Diagnostic{Code: "AF204", Severity: diag.Warning, Msg: "second"})

	if len(diags) != 2 {
		t.Fatalf("got %d diagnostics, want 2", len(diags))
	}
}

func TestRenderGolden(t *testing.T) {
	t.Parallel()

	source := "agent reviewer {\n  model: opus\n  out: Verdict\n}\n"

	diags := diag.Diagnostics{
		{
			Code:     "AF000",
			Severity: diag.Error,
			Msg:      "unexpected token ';'",
			Pos: lexer.Position{
				Filename: "examples/review.af",
				Line:     2,
				Column:   14,
			},
		},
		{
			Code:     "AF204",
			Severity: diag.Warning,
			Msg:      "branch may not cover all enum values",
			Pos: lexer.Position{
				Filename: "examples/review.af",
				Line:     3,
				Column:   3,
			},
		},
	}

	got := diag.Render(source, diags)
	want := "examples/review.af:2:14: error AF000: unexpected token ';'\n" +
		"  model: opus\n" +
		"             ^\n" +
		"examples/review.af:3:3: warning AF204: branch may not cover all enum values\n" +
		"  out: Verdict\n" +
		"  ^"

	if got != want {
		t.Fatalf("Render mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderEmpty(t *testing.T) {
	t.Parallel()

	if got := diag.Render("x", nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestHasSourceLocation(t *testing.T) {
	t.Parallel()

	withLine := diag.Diagnostic{
		Pos: lexer.Position{Filename: "f.af", Line: 1, Column: 1},
	}
	if !withLine.HasSourceLocation() {
		t.Fatal("expected HasSourceLocation true when Line > 0")
	}

	fileOnly := diag.Diagnostic{
		Pos: lexer.Position{Filename: "missing.af", Line: 0},
	}
	if fileOnly.HasSourceLocation() {
		t.Fatal("expected HasSourceLocation false when Line is 0")
	}
}

// Package pipeline is the single shared compile entry point that wires the
// AgentFlow stages — parse, resolve, inline/normalize, validate, IR — behind
// one function. The CLI, the end-to-end tests, and future tooling (LSP,
// simulator) all funnel through Compile so the stage sequence lives in exactly
// one place.
package pipeline

import (
	"os"
	"path/filepath"

	"github.com/alecthomas/participle/v2/lexer"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/flowgraph"
	"github.com/Abhinand20/agentFlow/internal/ir"
	"github.com/Abhinand20/agentFlow/internal/parser"
	"github.com/Abhinand20/agentFlow/internal/sema"
	"github.com/Abhinand20/agentFlow/internal/validate"
)

// Result carries the compiled IR together with the original source text so
// callers can render diagnostics (diag.Render needs the source) without a
// second read of the file.
type Result struct {
	IR     ir.Program
	Source string
}

// Compile runs parse -> resolve -> inline/normalize -> validate -> IR for the
// .af file at path. It returns the IR (zero value when there are errors) and
// ALL accumulated diagnostics, including warnings emitted on an otherwise clean
// program so callers can surface them and still exit 0.
func Compile(path string) (Result, diag.Diagnostics) {
	var diags diag.Diagnostics

	src, err := os.ReadFile(path)
	if err != nil {
		diags.Add(diag.Diagnostic{
			Code:     "AF000",
			Severity: diag.Error,
			Msg:      "cannot read source: " + err.Error(),
			Pos:      lexer.Position{Filename: path},
		})
		return Result{}, diags
	}
	source := string(src)
	result := Result{Source: source}

	astRoot, d := parser.Parse(path, source)
	diags.Add(d...)
	if diags.HasErrors() {
		return result, diags
	}

	srcDir := filepath.Dir(path)
	prog, d := sema.Resolve(astRoot, srcDir)
	diags.Add(d...)
	if diags.HasErrors() {
		return result, diags
	}

	res, d := flowgraph.Resolve(prog)
	diags.Add(d...)

	// Always run validate (even when flowgraph cut recursion) to surface as
	// many issues as possible, but do not build IR if anything errored.
	diags.Add(validate.Validate(prog, res)...)
	if diags.HasErrors() {
		return result, diags
	}

	program, d := ir.FromResolved(prog, res)
	diags.Add(d...)
	if diags.HasErrors() {
		return result, diags
	}

	result.IR = program
	return result, diags
}

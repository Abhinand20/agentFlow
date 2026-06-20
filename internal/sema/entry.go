package sema

import (
	"sort"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func accountEntryFlow(prog *model.Program, diags *diag.Diagnostics) {
	var entries []string
	for _, ref := range prog.Order {
		if ref.Kind != model.DeclFlow {
			continue
		}
		fl := prog.Flows[ref.Name]
		if fl != nil && fl.Entry {
			entries = append(entries, fl.Name)
		}
	}
	switch len(entries) {
	case 0:
		diags.Add(diag.Diagnostic{
			Code:     "AF138",
			Severity: diag.Error,
			Msg:      "no entry flow",
			Pos:      progPos(prog),
		})
	case 1:
		prog.EntryFlow = entries[0]
	default:
		sort.Strings(entries)
		diags.Add(diag.Diagnostic{
			Code:     "AF139",
			Severity: diag.Error,
			Msg:      "multiple entry flows: " + strings.Join(entries, ", "),
			Pos:      progPos(prog),
		})
	}
}

func progPos(prog *model.Program) lexer.Position {
	if len(prog.Order) == 0 {
		return lexer.Position{}
	}
	ref := prog.Order[0]
	switch ref.Kind {
	case model.DeclCapability:
		if c := prog.Capabilities[ref.Name]; c != nil {
			return c.Pos
		}
	case model.DeclFlow:
		if f := prog.Flows[ref.Name]; f != nil {
			return f.Pos
		}
	}
	return lexer.Position{}
}

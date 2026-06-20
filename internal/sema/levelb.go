package sema

import (
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/Abhinand20/agentFlow/internal/diag"
)

func emitLevelB(diags *diag.Diagnostics, pos lexer.Position, construct string) {
	diags.Add(diag.Diagnostic{
		Code:     "AF150",
		Severity: diag.Error,
		Msg:      construct + " unsupported in v0.1 (Level B)",
		Pos:      pos,
	})
}

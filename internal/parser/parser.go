package parser

import (
	"errors"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"

	"github.com/Abhinand20/agentFlow/internal/ast"
	"github.com/Abhinand20/agentFlow/internal/diag"
)

var parser = participle.MustBuild[ast.AST](
	participle.Lexer(afLexer),
	participle.Elide("Comment", "Whitespace"),
	participle.Unquote("String"),
	participle.UseLookahead(2),
)

// Parse turns AgentFlow source into an AST. On failure returns AF000 diagnostics.
func Parse(filename, src string) (*ast.AST, diag.Diagnostics) {
	root, err := parser.ParseString(filename, src)
	if err != nil {
		var diags diag.Diagnostics
		diags.Add(parseErrToDiag(err, filename))
		return nil, diags
	}
	return root, nil
}

func parseErrToDiag(err error, filename string) diag.Diagnostic {
	var pe participle.Error
	if errors.As(err, &pe) {
		pos := pe.Position()
		if pos.Filename == "" {
			pos.Filename = filename
		}
		return diag.Diagnostic{
			Code:     "AF000",
			Severity: diag.Error,
			Msg:      pe.Message(),
			Pos:      pos,
		}
	}
	return diag.Diagnostic{
		Code:     "AF000",
		Severity: diag.Error,
		Msg:      err.Error(),
		Pos:      lexer.Position{Filename: filename},
	}
}

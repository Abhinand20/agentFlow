package parser

import (
	"errors"
	"sync"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"

	"github.com/Abhinand20/agentFlow/internal/ast"
	"github.com/Abhinand20/agentFlow/internal/diag"
)

var (
	afParserOnce sync.Once
	afParser     *participle.Parser[ast.AST]
	afParserErr  error
)

func getParser() (*participle.Parser[ast.AST], error) {
	afParserOnce.Do(func() {
		afParser, afParserErr = participle.Build[ast.AST](
			participle.Lexer(afLexer),
			participle.Elide("Comment", "Whitespace"),
			participle.Unquote("String"),
			participle.UseLookahead(2),
		)
	})
	return afParser, afParserErr
}

// Parse turns AgentFlow source into an AST. On failure returns AF000 diagnostics.
func Parse(filename, src string) (*ast.AST, diag.Diagnostics) {
	p, err := getParser()
	if err != nil {
		var diags diag.Diagnostics
		diags.Add(diag.Diagnostic{
			Code:     "AF000",
			Severity: diag.Error,
			Msg:      err.Error(),
			Pos:      lexer.Position{Filename: filename},
		})
		return nil, diags
	}

	root, err := p.ParseString(filename, src)
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

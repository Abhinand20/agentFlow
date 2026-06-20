package parser

import (
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/Abhinand20/agentFlow/internal/ast"
)

func TestBuildGrammar(t *testing.T) {
	_, err := participle.Build[ast.AST](
		participle.Lexer(afLexer),
		participle.Elide("Comment", "Whitespace"),
		participle.Unquote("String"),
		participle.UseLookahead(2),
	)
	if err != nil {
		t.Fatalf("build AST: %v", err)
	}
}

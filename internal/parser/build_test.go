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

func TestBuildGrammarValue(t *testing.T) {
	_, err := participle.Build[ast.Value](
		participle.Lexer(afLexer),
		participle.Unquote("String"),
	)
	if err != nil {
		t.Fatalf("build Value: %v", err)
	}
}

func TestBuildGrammarFlow(t *testing.T) {
	_, err := participle.Build[ast.Flow](
		participle.Lexer(afLexer),
		participle.Unquote("String"),
		participle.UseLookahead(2),
	)
	if err != nil {
		t.Fatalf("build Flow: %v", err)
	}
}

func TestBuildGrammarLoop(t *testing.T) {
	_, err := participle.Build[ast.Loop](
		participle.Lexer(afLexer),
		participle.UseLookahead(2),
	)
	if err != nil {
		t.Fatalf("build Loop: %v", err)
	}
}

func TestBuildGrammarList(t *testing.T) {
	_, err := participle.Build[ast.List](
		participle.Lexer(afLexer),
		participle.Unquote("String"),
		participle.UseLookahead(2),
	)
	if err != nil {
		t.Fatalf("build List: %v", err)
	}
}

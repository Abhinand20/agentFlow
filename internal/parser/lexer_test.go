package parser

import (
	"strings"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
)

func lexAll(t *testing.T, src string) []lexer.Token {
	t.Helper()
	lex, err := afLexer.Lex("", strings.NewReader(src))
	if err != nil {
		t.Fatalf("Lex(%q): %v", src, err)
	}
	var tokens []lexer.Token
	for tok, err := lex.Next(); !tok.EOF(); tok, err = lex.Next() {
		if err != nil {
			t.Fatalf("Next(): %v", err)
		}
		if tok.Type == lexer.EOF {
			break
		}
		name := tokenTypeName(tok)
		if name == "Whitespace" || name == "Comment" {
			continue
		}
		tokens = append(tokens, tok)
	}
	return tokens
}

func tokenTypeName(tok lexer.Token) string {
	for name, tt := range afLexer.Symbols() {
		if tt == tok.Type {
			return name
		}
	}
	return "?"
}

func tokenTypes(tokens []lexer.Token) []string {
	types := make([]string, len(tokens))
	for i, tok := range tokens {
		types[i] = tokenTypeName(tok)
	}
	return types
}

func tokenValues(tokens []lexer.Token) []string {
	vals := make([]string, len(tokens))
	for i, tok := range tokens {
		vals[i] = tok.Value
	}
	return vals
}

func TestLexerChainArrow(t *testing.T) {
	t.Parallel()
	tokens := lexAll(t, "a->b")
	types := tokenTypes(tokens)
	want := []string{"Ident", "Op", "Ident"}
	if len(types) != len(want) {
		t.Fatalf("got %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("token[%d] type = %q, want %q", i, types[i], want[i])
		}
	}
	vals := tokenValues(tokens)
	if vals[0] != "a" || vals[1] != "->" || vals[2] != "b" {
		t.Fatalf("values = %v", vals)
	}
}

func TestLexerHyphenatedIdent(t *testing.T) {
	t.Parallel()
	tokens := lexAll(t, "on-fail prompt-file model-provider")
	types := tokenTypes(tokens)
	for i := range []string{"Ident", "Ident", "Ident"} {
		if types[i] != "Ident" {
			t.Fatalf("token[%d] = %q, want Ident", i, types[i])
		}
	}
}

func TestLexerQualName(t *testing.T) {
	t.Parallel()
	tokens := lexAll(t, "anthropic.opus")
	types := tokenTypes(tokens)
	want := []string{"Ident", "Punct", "Ident"}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("token[%d] = %q, want %q", i, types[i], want[i])
		}
	}
}

func TestLexerNotEqual(t *testing.T) {
	t.Parallel()
	tokens := lexAll(t, "review != revise")
	types := tokenTypes(tokens)
	want := []string{"Ident", "Op", "Ident"}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("token[%d] = %q, want %q", i, types[i], want[i])
		}
	}
	if tokenValues(tokens)[1] != "!=" {
		t.Fatalf("op value = %q", tokenValues(tokens)[1])
	}
}

func TestLexerStringWithEnvRef(t *testing.T) {
	t.Parallel()
	tokens := lexAll(t, `"with ${NAME}"`)
	if len(tokens) != 1 || tokenTypeName(tokens[0]) != "String" {
		t.Fatalf("got %v", tokens)
	}
	if tokens[0].Value != `"with ${NAME}"` {
		t.Fatalf("value = %q", tokens[0].Value)
	}
}

func TestLexerSemicolonError(t *testing.T) {
	t.Parallel()
	lex, err := afLexer.Lex("", strings.NewReader("a; b"))
	if err != nil {
		return // Lex-level error is acceptable
	}
	if _, err := lex.Next(); err != nil {
		return
	}
	if _, err := lex.Next(); err == nil {
		t.Fatal("expected lexer error for semicolon")
	}
}

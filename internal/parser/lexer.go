// Package parser turns AgentFlow (.af) source into a syntax tree.
//
// Keywords are not reserved at the lexer level — they lex as Ident and are matched
// positionally in the grammar. In step position, keyword-led productions are tried
// before bare refs, so an agent literally named loop/branch/etc. cannot be used as
// a bare step reference (accepted v0.1 limitation).
package parser

import "github.com/alecthomas/participle/v2/lexer"

// afLexer tokenizes AgentFlow source. Rule order matters: participle tries rules
// top-to-bottom and takes the first match at each position.
// AfLexer returns the AgentFlow lexer definition.
func AfLexer() lexer.Definition { return afLexer }

var afLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Comment", Pattern: `#[^\n]*`},
	{Name: "Whitespace", Pattern: `\s+`},
	{Name: "String", Pattern: `"(\\.|[^"\\])*"`},
	{Name: "Number", Pattern: `[0-9]+(\.[0-9]+)?`},
	{Name: "Op", Pattern: `->|==|!=`},
	{Name: "Ident", Pattern: `[a-zA-Z][a-zA-Z0-9_]*(-[a-zA-Z0-9_]+)*`},
	{Name: "Punct", Pattern: `[{}\[\]():,.|=]`},
})

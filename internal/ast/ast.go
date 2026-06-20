package ast

import "github.com/alecthomas/participle/v2/lexer"

// AST is the root of a parsed AgentFlow file.
type AST struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Decls []*Decl        `@@*`
}

// Decl is one top-level declaration.
type Decl struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Use   *Use           `  @@`
	Type  *TypeDecl      `| @@`
	Agent *Agent         `| @@`
	Gate  *Gate          `| @@`
	Flow  *Flow          `| @@`
}

// Use declares a capability (inline block or Level B alias form).
type Use struct {
	Pos    lexer.Position `parser:"" json:"-"`
	Name   *QualName      `"use" @@`
	Alias  *string        `parser:"( \"as\" @Ident "`
	Fields []*Field       `parser:"| \"{\" @@* \"}\" )?"`
}

// TypeDecl is an enum type declaration.
type TypeDecl struct {
	Pos    lexer.Position `parser:"" json:"-"`
	Name   string         `"type" @Ident "="`
	Values []string       `@Ident ( "|" @Ident )*`
}

// Agent is a generic field bag; M2 interprets fields by key.
type Agent struct {
	Pos    lexer.Position `parser:"" json:"-"`
	Name   string         `"agent" @Ident`
	Fields []*Field       `"{" @@* "}"`
}

// Gate is a generic field bag; M2 interprets fields by key.
type Gate struct {
	Pos    lexer.Position `parser:"" json:"-"`
	Name   string         `"gate" @Ident`
	Fields []*Field       `"{" @@* "}"`
}

// Flow is a named flow with optional entry marker and Level B params.
type Flow struct {
	Pos    lexer.Position `parser:"" json:"-"`
	Entry  bool           `@"entry"?`
	Name   string         `"flow" @Ident`
	Params *Params        `@@?`
	Items  []*FlowItem    `"{" @@* "}"`
}

// FlowItem is either a header field or a step (disambiguated by lookahead on ":").
type FlowItem struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Field *Field         `  @@`
	Step  *Step          `| @@`
}

// Field is a key: value pair inside a block.
type Field struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Key   string         `@Ident ":"`
	Value *Value         `@@`
}

// Value is a field value (bool, string, number, list, or qual-name ref).
type Value struct {
	Pos  lexer.Position `parser:"" json:"-"`
	Bool *bool          `parser:"( @('true' | 'false') "`
	Str  *string        `parser:"| @String "`
	Num  *float64       `parser:"| @Number "`
	List *List          `parser:"| @@ "`
	Ref  *QualName      `parser:"| @@ )"`
}

// ListElem is a value inside a list (no nested lists — avoids parser grammar cycles).
type ListElem struct {
	Pos  lexer.Position `parser:"" json:"-"`
	Bool *bool          `parser:"( @('true' | 'false') "`
	Str  *string        `parser:"| @String "`
	Num  *float64       `parser:"| @Number "`
	Ref  *QualName      `parser:"| @@ )"`
}

// List is a bracketed list of values.
type List struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Items []*ListElem    `'[' @@ ( ',' @@ )* ']'`
}

// QualName is a dotted identifier (one or more parts).
type QualName struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Parts []string       `@Ident ( "." @Ident )*`
}

// Step is one flow step (keyword-led forms before ident-led Chain).
type Step struct {
	Pos      lexer.Position `parser:"" json:"-"`
	Parallel *Parallel      `  @@`
	Branch   *Branch        `| @@`
	Loop     *Loop          `| @@`
	Repeat   *Repeat        `| @@`
	Chain    *Chain         `| @@`
}

// Chain is a sequence of atoms connected by ->.
type Chain struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Atoms []*Atom        `@@ ( "->" @@ )*`
	Attr  *EdgeAttr      `@@?`
}

// Atom is a ref, call, or gather target with optional alias.
type Atom struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Name  *QualName      `@@`
	Args  *Args          `( "(" @@? ")" )?`
	Block []*Step        `( "{" @@* "}" )?`
	Alias *string        `( "as" @Ident )?`
}

// Args holds comma-separated call arguments (Level B).
type Args struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Items []*Value       `@@ ( "," @@ )*`
}

// EdgeAttr is a Level B edge attribute on a chain.
type EdgeAttr struct {
	Pos  lexer.Position `parser:"" json:"-"`
	When *Cond          `"[" ( "when" @@ )?`
	Max  *float64       `( ","? "max" @Number )? "]"`
}

// Branch selects on a value label's enum output.
type Branch struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Value *ValueRef      `"branch" @@`
	Cases []*Case        `"{" @@* "}"`
}

// Case is one branch arm.
type Case struct {
	Pos    lexer.Position `parser:"" json:"-"`
	Values []string       `"case" @Ident ( "," @Ident )*`
	Step   *Step          `"->" @@`
}

// ValueRef names a value label or Level B "it".
type ValueRef struct {
	Pos  lexer.Position `parser:"" json:"-"`
	It   bool           `parser:"( @\"it\" "`
	Name *QualName      `parser:"| @@ )"`
}

// Loop repeats a body while a condition holds (until before body).
type Loop struct {
	Pos  lexer.Position `parser:"" json:"-"`
	Cond *Cond          `"loop" "(" ( "until" @@ )?`
	Max  *float64       `( ","? "max" @Number )? ")"`
	Body []*Step        `"{" @@* "}"`
}

// Repeat runs a body at least once (until after body, do-while).
type Repeat struct {
	Pos  lexer.Position `parser:"" json:"-"`
	Body []*Step        `"repeat" "{" @@* "}"`
	Cond *Cond          `"until" "(" @@`
	Max  *float64       `( ","? "max" @Number )? ")"`
}

// Cond is a flat enum comparison on a value label.
type Cond struct {
	Pos   lexer.Position `parser:"" json:"-"`
	Value *ValueRef      `@@`
	Op    string         `@( "==" | "!=" )`
	Enum  string         `@Ident`
}

// Parallel runs branches concurrently with optional gather.
type Parallel struct {
	Pos    lexer.Position `parser:"" json:"-"`
	Each   *Each          `"parallel" ( @@ )?`
	Body   []*Step        `"{" @@* "}"`
	Gather *Atom          `( "gather" @@ )?`
}

// Each is Level B dynamic fan-out inside parallel.
type Each struct {
	Pos  lexer.Position `parser:"" json:"-"`
	Item *QualName      `"each" @@`
	As   string         `"as" @Ident`
}

// Params is Level B flow parameter list.
type Params struct {
	Pos    lexer.Position `parser:"" json:"-"`
	Params []*Param       `"(" @@ ( "," @@ )* ")"`
}

// Param is one flow parameter with optional type annotation.
type Param struct {
	Pos  lexer.Position `parser:"" json:"-"`
	Name string         `@Ident`
	Type *string        `( ":" @Ident )?`
}
